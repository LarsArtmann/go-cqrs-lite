package irohengine

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Transport carries CRDT-safe write operations between nodes.
// In production this would be backed by Iroh's iroh-docs CRDT sync;
// the in-process Network implementation simulates P2P delivery for testing.
type Transport interface {
	Publish(ctx context.Context, op WriteOp) error
	Subscribe(handler func(op WriteOp)) error
	Close() error
}

// NetworkOption configures a Network.
type NetworkOption func(*Network)

// WithNetworkDelay injects a random latency into message delivery,
// simulating real P2P network conditions. Zero means synchronous delivery.
// The actual delivery time is MEASURED per-message — this configures the
// simulated link, not the reported stats.
func WithNetworkDelay(max time.Duration) NetworkOption {
	return func(n *Network) {
		n.maxDelay = max
	}
}

// WithNetworkDropRate sets the probability [0.0–1.0] that a message is dropped,
// simulating unreliable network conditions. Zero means no drops.
func WithNetworkDropRate(rate float64) NetworkOption {
	return func(n *Network) {
		n.dropRate = rate
	}
}

// Network simulates a P2P network in-process. Multiple engines join the same
// Network to sync writes. Messages are delivered in parallel to all peers via
// goroutines, and real delivery/convergence times are measured per message.
type Network struct {
	mu        sync.RWMutex
	peers     map[string]*peerTransport
	maxDelay  time.Duration
	dropRate  float64
	closed    bool
	collector *LatencyCollector
}

// NewNetwork creates an in-process P2P network simulator with real latency measurement.
func NewNetwork(opts ...NetworkOption) *Network {
	n := &Network{
		peers:     make(map[string]*peerTransport),
		collector: newLatencyCollector(),
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// Collector returns the network's latency collector for direct stats access.
func (n *Network) Collector() *LatencyCollector {
	return n.collector
}

// Join registers a new node on the network and returns its Transport.
func (n *Network) Join(nodeID string) Transport {
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

func (n *Network) getOtherPeers(from string) []*peerTransport {
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
func (n *Network) deliver(from string, op WriteOp) {
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
		if n.dropRate > 0 && rand.Float64() < n.dropRate {
			continue
		}

		wg.Add(1)
		go func(p *peerTransport) {
			defer wg.Done()

			if n.maxDelay > 0 {
				time.Sleep(time.Duration(rand.Int63n(int64(n.maxDelay))))
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

func (n *Network) close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
}

// peerTransport is a per-node Transport backed by a Network.
type peerTransport struct {
	nodeID  string
	network *Network
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
