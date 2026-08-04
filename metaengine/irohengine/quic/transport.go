//go:build cgo

package quic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// DefaultALPN is the Application-Layer Protocol Negotiation bytes used by all
// QuicTransport endpoints. All nodes must share the same ALPN to connect.
var DefaultALPN = []byte("irohengine/crdt/v1") //nolint:gochecknoglobals // configurable default

// maxOpSize is the maximum serialized WriteOp size accepted over a QUIC stream.
// 16 MB is generous for CRDT ops while preventing memory exhaustion.
const maxOpSize = 16 * 1024 * 1024

// rttWindowSize is how many RTT samples we keep for percentile computation.
const rttWindowSize = 256

// QuicTransport implements irohengine.Transport over real Iroh QUIC streams.
//
// This is REAL networking: every Publish opens a QUIC BiStream, serializes the
// WriteOp, sends it over the wire, and the receiver deserializes and dispatches
// it. Latency is measured from QUIC's own ACK timing via conn.Rtt().
//
// Requires CGo (links the Iroh Rust static library via iroh-go).
type QuicTransport struct {
	endpoint *iroh_ffi.Endpoint
	alpn     []byte
	cfg      *config

	mu     sync.RWMutex
	conns  map[string]*peerConn // peerID string → connection
	subs   []func(op irohengine.WriteOp)
	closed bool

	// RTT measurement (real QUIC ACK timing)
	rttMu      sync.Mutex
	rttSamples []time.Duration

	// Op-level dedup (prevents double-application of non-idempotent ops
	// like SetAdd/CounterIncrement under redelivery or relay echo)
	dedupMu   sync.Mutex
	dedupSeen map[string]struct{}

	acceptWG sync.WaitGroup
}

// peerConn tracks a single QUIC connection and its metadata.
type peerConn struct {
	conn   *iroh_ffi.Connection
	peerID string
}

var errQuicTransportClosed = errors.New("transport closed")

// New creates a QuicTransport by binding a real Iroh QUIC endpoint.
// The endpoint starts listening immediately; use Ticket() to get the
// connection string for other nodes to connect.
func New(opts ...Option) (*QuicTransport, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	preset := cfg.presetFn()
	bindAddr := cfg.bindAddr

	ep, err := iroh_ffi.EndpointBind(iroh_ffi.EndpointOptions{
		Preset:   &preset,
		BindAddr: &bindAddr,
		Alpns:    &[][]byte{cfg.alpn},
	})
	if err != nil {
		return nil, fmt.Errorf("iroh endpoint bind failed: %w", err)
	}

	t := &QuicTransport{
		endpoint:  ep,
		alpn:      cfg.alpn,
		cfg:       cfg,
		conns:     make(map[string]*peerConn),
		dedupSeen: make(map[string]struct{}),
	}

	// Start accept loop
	t.acceptWG.Add(1)
	go t.acceptLoop()

	return t, nil
}

// Ticket returns the base32 connection ticket for this endpoint.
// Share this string with other nodes so they can Connect().
func (t *QuicTransport) Ticket() (string, error) {
	t.mu.RLock()
	closed := t.closed
	t.mu.RUnlock()
	if closed {
		return "", errQuicTransportClosed
	}

	ticket, err := iroh_ffi.EndpointTicketFromAddr(t.endpoint.Addr())
	if err != nil {
		return "", fmt.Errorf("failed to generate ticket: %w", err)
	}
	return ticket.String(), nil
}

// NodeID returns this endpoint's unique identifier (ed25519 public key hex).
func (t *QuicTransport) NodeID() string {
	return t.endpoint.Id().String()
}

// PeerCount returns the number of currently connected peers.
// Useful for tests to wait until connections are established.
func (t *QuicTransport) PeerCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.conns)
}

// Connect dials a remote endpoint using its ticket string and establishes
// a QUIC connection. The remote endpoint's accept loop will register the
// connection automatically.
func (t *QuicTransport) Connect(ticketStr string) error {
	t.mu.RLock()
	closed := t.closed
	t.mu.RUnlock()
	if closed {
		return errQuicTransportClosed
	}

	ticket, err := iroh_ffi.EndpointTicketFromString(ticketStr)
	if err != nil {
		return fmt.Errorf("invalid ticket: %w", err)
	}
	addr := ticket.EndpointAddr()

	conn, err := t.endpoint.Connect(addr, t.alpn)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	peerID := conn.RemoteId().String()
	t.mu.Lock()
	t.conns[peerID] = &peerConn{conn: conn, peerID: peerID}
	t.mu.Unlock()

	// Start handling incoming streams from this connection
	go t.handleConnection(conn, peerID)

	return nil
}

// Publish implements irohengine.Transport. It serializes the WriteOp and sends
// it over a real QUIC BiStream to every connected peer. Blocks until all peers
// have received the data at the QUIC level.
func (t *QuicTransport) Publish(_ context.Context, op irohengine.WriteOp) error {
	data, err := encodeOp(op)
	if err != nil {
		return fmt.Errorf("encode op: %w", err)
	}

	t.mu.RLock()
	conns := make([]*iroh_ffi.Connection, 0, len(t.conns))
	for _, pc := range t.conns {
		conns = append(conns, pc.conn)
	}
	t.mu.RUnlock()

	if len(conns) == 0 {
		return nil // no peers — local-only write
	}

	var wg sync.WaitGroup
	for _, conn := range conns {
		wg.Add(1)
		go func(c *iroh_ffi.Connection) {
			defer wg.Done()
			t.sendOp(c, data)
		}(conn)
	}
	wg.Wait()

	return nil
}

func (t *QuicTransport) sendOp(conn *iroh_ffi.Connection, data []byte) {
	stream, err := conn.OpenBi()
	if err != nil {
		return
	}
	_ = stream.Send().WriteAll(data)
	_ = stream.Send().Finish()

	// Record real RTT from QUIC's ACK timing
	if rtt := conn.Rtt(); rtt != nil {
		t.recordRTT(time.Duration(*rtt))
	}

	// Read response (empty ack) to complete the stream cleanly
	_, _ = stream.Recv().ReadToEnd(1024)
}

// Subscribe implements irohengine.Transport. Registers a handler that is
// called for every WriteOp received over QUIC from any peer.
func (t *QuicTransport) Subscribe(handler func(op irohengine.WriteOp)) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subs = append(t.subs, handler)
	return nil
}

// Close implements irohengine.Transport. Closes all connections and the endpoint.
func (t *QuicTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	conns := make([]*iroh_ffi.Connection, 0, len(t.conns))
	for _, pc := range t.conns {
		conns = append(conns, pc.conn)
	}
	t.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close(0, []byte("shutdown"))
	}
	_ = t.endpoint.Close()

	// Wait for accept loop to finish
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

// Compile-time assertion that QuicTransport implements irohengine.Transport.
var _ irohengine.Transport = (*QuicTransport)(nil)
