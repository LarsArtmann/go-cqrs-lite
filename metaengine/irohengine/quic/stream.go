//go:build cgo

package quic

import (
	"time"

	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

func (t *QuicTransport) acceptLoop() {
	defer t.acceptWG.Done()
	for {
		incomingPtr := t.endpoint.AcceptNext()
		if incomingPtr == nil || *incomingPtr == nil {
			return // endpoint closed
		}

		accepting, err := (*incomingPtr).Accept()
		if err != nil {
			continue
		}
		conn, err := accepting.Connect()
		if err != nil {
			continue
		}

		peerID := conn.RemoteId().String()

		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			_ = conn.Close(0, []byte("closing"))
			return
		}
		t.conns[peerID] = &peerConn{conn: conn, peerID: peerID}
		t.mu.Unlock()

		go t.handleConnection(conn, peerID)
	}
}

func (t *QuicTransport) handleConnection(conn *iroh_ffi.Connection, peerID string) {
	for {
		stream, err := conn.AcceptBi()
		if err != nil {
			return // connection closed
		}
		go t.handleStream(conn, peerID, stream)
	}
}

func (t *QuicTransport) handleStream(
	conn *iroh_ffi.Connection,
	sourcePeerID string,
	stream *iroh_ffi.BiStream,
) {
	data, err := stream.Recv().ReadToEnd(maxOpSize)
	if err != nil {
		return
	}

	// Send empty ack so sender's ReadToEnd completes
	_ = stream.Send().WriteAll([]byte{})
	_ = stream.Send().Finish()

	op, err := decodeOp(data)
	if err != nil {
		return
	}

	// Op-level dedup: skip if already seen.
	if !t.markSeen(op.ID) {
		return
	}

	// Record real RTT from QUIC's ACK timing
	if rtt := conn.Rtt(); rtt != nil {
		t.recordRTT(time.Duration(*rtt))
	}

	// Dispatch to local subscribers
	t.mu.RLock()
	subs := t.subs
	t.mu.RUnlock()
	for _, s := range subs {
		s(op)
	}

	// Relay: forward to all OTHER peers (star topology support)
	if t.cfg.relay {
		t.relayToOthers(sourcePeerID, op)
	}
}

// markSeen records opID as seen, returning false if it was already seen.
// The seen-set is bounded; when it exceeds 10K entries it resets.
func (t *QuicTransport) markSeen(opID string) bool {
	t.dedupMu.Lock()
	defer t.dedupMu.Unlock()
	if _, seen := t.dedupSeen[opID]; seen {
		return false
	}
	t.dedupSeen[opID] = struct{}{}
	if len(t.dedupSeen) > 10000 {
		t.dedupSeen = make(map[string]struct{})
		t.dedupSeen[opID] = struct{}{}
	}
	return true
}

// relayToOthers forwards an op to all connected peers except the source.
func (t *QuicTransport) relayToOthers(sourcePeerID string, op irohengine.WriteOp) {
	data, err := encodeOp(op)
	if err != nil {
		return
	}

	t.mu.RLock()
	var targets []*iroh_ffi.Connection
	for id, pc := range t.conns {
		if id != sourcePeerID {
			targets = append(targets, pc.conn)
		}
	}
	t.mu.RUnlock()

	for _, conn := range targets {
		go func(c *iroh_ffi.Connection) {
			t.sendOp(c, data)
		}(conn)
	}
}
