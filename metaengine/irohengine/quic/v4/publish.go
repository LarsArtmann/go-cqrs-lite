//go:build cgo

package quic

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

// Publish implements irohengine.Transport. It serializes the WriteOp and sends
// it over a real QUIC BiStream to every connected peer. Blocks until all peers
// have received the data at the QUIC level.
//
// Cancelling ctx ends the wait: the connections whose send has not completed
// are closed (so their blocking stream calls return and the peers are evicted
// from the connection map) and the context's error is returned.
func (t *QuicTransport) Publish(ctx context.Context, op irohengine.WriteOp) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("quic.Publish: %w", err)
	}

	data, err := encodeOp(op)
	if err != nil {
		return fmt.Errorf("encode op: %w", err)
	}

	t.mu.RLock()
	peers := slices.Collect(maps.Values(t.conns))
	t.mu.RUnlock()

	if len(peers) == 0 {
		return nil // no peers — local-only write
	}

	pending := newPendingPeers(peers)
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			pending.closeConnections()
		case <-done:
		}
	}()

	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)

		go func(peer *peerConn) {
			defer wg.Done()
			defer pending.finished(peer)

			if t.cfg.poolStreams {
				t.sendOpPooled(peer, data)
			} else {
				t.sendOp(peer.conn, data)
			}
		}(peer)
	}

	wg.Wait()
	close(done)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("quic.Publish: %w", err)
	}

	return nil
}

// pendingPeers tracks the peers one Publish is still sending to. The sends
// hold no context-aware primitive (a pooled send blocks in an ack read with
// no deadline), so cancelling has to act on the connection itself — and only
// for sends still in flight, never for peers that already received the op.
type pendingPeers struct {
	mu    sync.Mutex
	peers map[*peerConn]struct{}
}

func newPendingPeers(peers []*peerConn) *pendingPeers {
	p := &pendingPeers{peers: make(map[*peerConn]struct{}, len(peers))}

	for _, peer := range peers {
		p.peers[peer] = struct{}{}
	}

	return p
}

func (p *pendingPeers) finished(peer *peerConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.peers, peer)
}

// closeConnections closes every connection whose send has not completed.
func (p *pendingPeers) closeConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for peer := range p.peers {
		_ = peer.conn.Close(0, []byte("publish cancelled"))
	}
}

var _ irohengine.Transport = (*QuicTransport)(nil)
