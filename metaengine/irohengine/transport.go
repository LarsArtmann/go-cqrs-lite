package irohengine

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Transport carries CRDT-safe write operations between nodes.
// Implementations:
//   - InProcessNetwork: goroutine-based delivery for unit tests (no real networking)
//   - quic.QuicTransport: real QUIC streams via iroh-go (requires CGo)
type Transport interface {
	Publish(ctx context.Context, op WriteOp) error
	Subscribe(handler func(op WriteOp)) error
	Close() error
}

// InProcessNetworkOption configures an InProcessNetwork.
type InProcessNetworkOption func(*InProcessNetwork)

// WithNetworkDelay injects a random latency into message delivery,
// simulating real P2P network conditions. Zero means synchronous delivery.
// The actual delivery time is MEASURED per-message — this configures the
// simulated link, not the reported stats.
func WithNetworkDelay(max time.Duration) InProcessNetworkOption {
	return func(n *InProcessNetwork) {
		n.maxDelay = max
	}
}

// WithNetworkDropRate sets the probability [0.0–1.0] that a message is dropped,
// simulating unreliable network conditions. Zero means no drops.
func WithNetworkDropRate(rate float64) InProcessNetworkOption {
	return func(n *InProcessNetwork) {
		n.dropRate = rate
	}
}

// InProcessNetwork simulates a P2P network entirely in-process using goroutines.
// This is a TEST HELPER — it uses function calls, not real networking.
// For real QUIC transport, use quic.QuicTransport (requires CGo + iroh-go).
//
// Messages are delivered in parallel to all peers via goroutines, and real
// delivery/convergence times are measured per message (useful for CI without CGo).
type InProcessNetwork struct {
	mu        sync.RWMutex
	peers     map[string]*peerTransport
	maxDelay  time.Duration
	dropRate  float64
	closed    bool
	collector *LatencyCollector
}

// NewInProcessNetwork creates an in-process P2P network simulator.
// This is for unit tests only — it delivers messages via goroutine function
// calls, not real network I/O. For real networking, use quic.NewTransport.
func NewInProcessNetwork(opts ...InProcessNetworkOption) *InProcessNetwork {
	n := &InProcessNetwork{
		peers:     make(map[string]*peerTransport),
		collector: newLatencyCollector(),
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// NewNetwork creates an in-process P2P network simulator.
// Deprecated: use NewInProcessNetwork for clarity. This alias is kept for
// backward compatibility with existing test code.
func NewNetwork(opts ...InProcessNetworkOption) *InProcessNetwork {
	return NewInProcessNetwork(opts...)
}

// Collector returns the network's latency collector for direct stats access.
func (n *InProcessNetwork) Collector() *LatencyCollector {
	return n.collector
}

// Join registers a new node on the network and returns its Transport.
func (n *InProcessNetwork) Join(nodeID string) Transport {
	n.mu.Lock()
	defer n.mu.Unlock()

	pt := &peerTransport{
		nodeID:  nodeID,
		network: n,
		subs:    make([]func(op WriteOp), 0),
	}
	n.peers[nodeID] = pt
	return pt
}

func (n *InProcessNetwork) getOtherPeers(from string) []*peerTransport {
	n.mu.RLock()
	defer n.mu.RUnlock()
	peers := make([]*peerTransport, 0, len(n.peers)-1)
	for _, p := range n.peers {
		if p.nodeID != from {
			peers = append(peers, p)
		}
	}
	return peers
}

// deliver sends an op to all peers in parallel, measuring real delivery
// and convergence times. Blocks until all peers have processed the op
// (or the network is closed).
//
//nolint:gosec // G404: math/rand is fine for simulated delay — not security-sensitive
func (n *InProcessNetwork) deliver(from string, op WriteOp) {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return
	}
	n.mu.RUnlock()

	peers := n.getOtherPeers(from)
	if len(peers) == 0 {
		return
	}

	var wg sync.WaitGroup
	var maxLatency atomic.Int64

	for _, p := range peers {
		if n.dropRate > 0 && rand.Float64() < n.dropRate { //nolint:gosec // G404: not security-sensitive
			continue
		}

		wg.Add(1)
		go func(p *peerTransport) {
			defer wg.Done()

			if n.maxDelay > 0 {
				time.Sleep(time.Duration(rand.Int63n(int64(n.maxDelay)))) //nolint:gosec // G404: not security-sensitive
			}

			deliveryLatency := time.Since(op.PublishedAt)
			n.collector.recordDelivery(deliveryLatency)

			p.deliver(op)

			applyLatency := int64(time.Since(op.PublishedAt))
			for {
				cur := maxLatency.Load()
				if applyLatency <= cur || maxLatency.CompareAndSwap(cur, applyLatency) {
					break
				}
			}
		}(p)
	}

	wg.Wait()

	if max := maxLatency.Load(); max > 0 {
		n.collector.recordConvergence(time.Duration(max))
	}
}

func (n *InProcessNetwork) close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
}

// peerTransport is a per-node Transport backed by an InProcessNetwork.
type peerTransport struct {
	nodeID  string
	network *InProcessNetwork
	mu      sync.RWMutex
	subs    []func(op WriteOp)
	closed  bool
}

func (pt *peerTransport) Publish(_ context.Context, op WriteOp) error {
	if op.PublishedAt.IsZero() {
		op.PublishedAt = time.Now()
	}
	pt.network.deliver(pt.nodeID, op)
	return nil
}

func (pt *peerTransport) Subscribe(handler func(op WriteOp)) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.subs = append(pt.subs, handler)
	return nil
}

func (pt *peerTransport) Close() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.closed = true
	return nil
}

// LatencySnapshot implements LatencyProvider so the engine can report
// real measured latency in its EngineProfile.
func (pt *peerTransport) LatencySnapshot() LatencySnapshot {
	return pt.network.collector.Snapshot()
}

func (pt *peerTransport) deliver(op WriteOp) {
	pt.mu.RLock()
	subs := pt.subs
	pt.mu.RUnlock()
	for _, s := range subs {
		s(op)
	}
}
