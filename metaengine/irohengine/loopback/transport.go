// Package loopback implements a TCP-based Transport for irohengine.
//
// This is the MIDDLE tier of the transport testing pyramid:
//   - InProcessNetwork: goroutine calls (no networking, no serialization)
//   - LoopbackTransport: real TCP connections + serialization (NO CGo required)
//   - QuicTransport: real QUIC + NAT traversal + ACK-based RTT (requires CGo)
//
// LoopbackTransport exercises real network code paths (length-prefix framing,
// partial reads, connection lifecycle, serialization round-trips) without any
// C dependencies. It is ideal for CI environments that lack a Rust toolchain
// but need to verify transport-layer correctness beyond what InProcessNetwork
// can test.
//
// Latency is measured from timestamps embedded in each frame, not from TCP
// ACK timing (TCP does not expose this). This gives real one-way delivery
// measurements suitable for convergence analysis.
package loopback

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// maxOpSize is the maximum serialized WriteOp size accepted over a TCP connection.
const maxOpSize = 16 * 1024 * 1024 // 16 MB

// frameHeaderSize is the number of bytes used for the length-prefix header.
const frameHeaderSize = 4

// rttWindowSize is how many latency samples we keep for percentile computation.
const rttWindowSize = 256

// LoopbackTransport implements irohengine.Transport over real TCP connections.
//
// Unlike InProcessNetwork (goroutine calls) and QuicTransport (QUIC via CGo),
// this transport uses Go's standard net package to create real TCP connections
// between nodes. Messages are framed with a 4-byte big-endian length prefix.
//
// This transport catches bugs that InProcessNetwork structurally cannot:
//   - Serialization round-trip issues (JSON encode/decode of WriteOp)
//   - Length-prefix framing bugs (partial reads, message boundary errors)
//   - Connection lifecycle bugs (accept loop, concurrent read/write, close)
//   - Real goroutine scheduling effects on message ordering
type LoopbackTransport struct {
	addr     string
	listener net.Listener

	mu     sync.RWMutex
	conns  map[string]net.Conn // peerAddr → connection
	subs   []func(op irohengine.WriteOp)
	closed bool

	// Latency measurement (real one-way delivery times)
	latencyMu sync.Mutex
	latencyMs []time.Duration

	// Op-level dedup (prevents double-application under redelivery)
	dedupMu   sync.Mutex
	dedupSeen map[string]struct{}

	// Optional simulated latency (for testing convergence under delay)
	maxDelay time.Duration

	acceptWG sync.WaitGroup
}

// errTransportClosed is returned when an operation is attempted on a closed transport.
var errTransportClosed = errors.New("transport closed")

// Option configures a LoopbackTransport.
type Option func(*config)

type config struct {
	addr     string
	maxDelay time.Duration
}

// WithAddr sets the bind address for the TCP listener.
// Default is "127.0.0.1:0" (localhost, random port).
func WithAddr(addr string) Option {
	return func(c *config) { c.addr = addr }
}

// WithSimulatedDelay injects random latency [0, maxDelay) into message delivery.
// useful for testing convergence behavior under network delay.
// Zero means no delay (messages delivered as fast as TCP allows).
func WithSimulatedDelay(maxDelay time.Duration) Option {
	return func(c *config) { c.maxDelay = maxDelay }
}

// New creates a LoopbackTransport by binding a real TCP listener.
func New(opts ...Option) (*LoopbackTransport, error) {
	cfg := &config{
		addr: "127.0.0.1:0",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", cfg.addr)
	if err != nil {
		return nil, fmt.Errorf("loopback listen failed: %w", err)
	}

	t := &LoopbackTransport{
		addr:      listener.Addr().String(),
		listener:  listener,
		conns:     make(map[string]net.Conn),
		dedupSeen: make(map[string]struct{}),
		maxDelay:  cfg.maxDelay,
	}

	t.acceptWG.Add(1)
	go t.acceptLoop()

	return t, nil
}

// Addr returns the transport's listen address (host:port).
func (t *LoopbackTransport) Addr() string {
	return t.addr
}

// Connect dials a remote LoopbackTransport at the given address.
func (t *LoopbackTransport) Connect(addr string) error {
	t.mu.RLock()
	closed := t.closed
	t.mu.RUnlock()
	if closed {
		return errTransportClosed
	}

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("loopback connect failed: %w", err)
	}

	t.mu.Lock()
	t.conns[addr] = conn
	t.mu.Unlock()

	go t.handleConnection(conn)

	return nil
}

// PeerCount returns the number of currently connected peers.
func (t *LoopbackTransport) PeerCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.conns)
}

// Publish implements irohengine.Transport. It serializes the WriteOp and sends
// it over real TCP connections to every connected peer.
func (t *LoopbackTransport) Publish(_ context.Context, op irohengine.WriteOp) error {
	if op.PublishedAt.IsZero() {
		op.PublishedAt = time.Now()
	}

	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("encode op: %w", err)
	}

	t.mu.RLock()
	conns := make([]net.Conn, 0, len(t.conns))
	for _, c := range t.conns {
		conns = append(conns, c)
	}
	t.mu.RUnlock()

	if len(conns) == 0 {
		return nil
	}

	for _, conn := range conns {
		if err := writeFrame(conn, data); err != nil {
			continue // peer may have disconnected
		}
	}

	return nil
}

// Subscribe implements irohengine.Transport.
func (t *LoopbackTransport) Subscribe(handler func(op irohengine.WriteOp)) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subs = append(t.subs, handler)
	return nil
}

// Close implements irohengine.Transport. Closes all connections and the listener.
func (t *LoopbackTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	conns := make([]net.Conn, 0, len(t.conns))
	for _, c := range t.conns {
		conns = append(conns, c)
	}
	t.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}
	_ = t.listener.Close()

	done := make(chan struct{})
	go func() {
		t.acceptWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	return nil
}

// LatencySnapshot implements irohengine.LatencyProvider, returning real
// one-way delivery latency measured from embedded timestamps.
func (t *LoopbackTransport) LatencySnapshot() irohengine.LatencySnapshot {
	t.latencyMu.Lock()
	samples := append([]time.Duration(nil), t.latencyMs...)
	t.latencyMu.Unlock()

	if len(samples) == 0 {
		return irohengine.LatencySnapshot{}
	}

	sorted := sortDurations(samples)
	p50 := sorted[len(sorted)/2]
	return irohengine.LatencySnapshot{
		DeliveryP50:    p50,
		DeliveryP99:    sorted[percentileIdx(len(sorted), 0.99)],
		ConvergenceP99: sorted[percentileIdx(len(sorted), 0.99)],
	}
}

// Compile-time assertion.
var _ irohengine.Transport = (*LoopbackTransport)(nil)
