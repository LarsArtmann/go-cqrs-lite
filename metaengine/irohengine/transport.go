package irohengine

import (
	"context"
	"math/rand"
	"sync"
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
// Network to sync writes. Messages are delivered asynchronously via goroutines.
type Network struct {
	mu       sync.RWMutex
	peers    map[string]*peerTransport
	maxDelay time.Duration
	dropRate float64
	closed   bool
}

// NewNetwork creates an in-process P2P network simulator.
func NewNetwork(opts ...NetworkOption) *Network {
	n := &Network{
		peers: make(map[string]*peerTransport),
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
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

func (n *Network) deliver(from string, op WriteOp) {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return
	}
	peers := make([]*peerTransport, 0, len(n.peers))
	for _, p := range n.peers {
		if p.nodeID == from {
			continue
		}
		peers = append(peers, p)
	}
	n.mu.RUnlock()

	if n.dropRate > 0 && rand.Float64() < n.dropRate {
		return
	}

	if n.maxDelay > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(n.maxDelay))))
	}

	for _, p := range peers {
		p.deliver(op)
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

func (pt *peerTransport) deliver(op WriteOp) {
	pt.mu.RLock()
	subs := pt.subs
	pt.mu.RUnlock()
	for _, s := range subs {
		s(op)
	}
}
