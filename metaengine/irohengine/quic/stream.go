//go:build cgo

package quic

import (
	"log/slog"
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
		if t.cfg.poolStreams {
			go t.handlePooledStream(conn, peerID, stream)
		} else {
			go t.handleStream(conn, peerID, stream)
		}
	}
}

func (t *QuicTransport) handleStream(
	conn *iroh_ffi.Connection,
	sourcePeerID string,
	stream *iroh_ffi.BiStream,
) {
	// Peek at the first byte to detect protocol mismatch: a pooled sender
	// writes a magic byte before any framing. If we see it here, the sender
	// is pooled but we are not — return immediately instead of hanging in
	// ReadToEnd waiting for a Finish() that never comes.
	firstByte, err := stream.Recv().ReadExact(1)
	if err != nil {
		return
	}

	if firstByte[0] == pooledStreamMagic {
		slog.Error("quic handleStream: protocol mismatch — pooled sender connected " +
			"to non-pooled receiver; enable WithStreamPooling on both nodes")
		_ = stream.Send().Finish() // unblock sender's ReadExact so it doesn't hang
		return
	}

	// Non-pooled sender: read the rest of the op data and combine with first byte
	rest, err := stream.Recv().ReadToEnd(maxOpSize)
	if err != nil {
		return
	}
	data := make([]byte, 0, 1+len(rest))
	data = append(data, firstByte...)
	data = append(data, rest...)

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
// Uses dedup.Ring for bounded memory with graceful eviction of the oldest
// entries, avoiding the reset-gap vulnerability of a fixed-size map.
func (t *QuicTransport) markSeen(opID string) bool {
	t.dedupMu.Lock()
	defer t.dedupMu.Unlock()
	if t.dedupRing.Has(opID) {
		return false
	}
	t.dedupRing.Add(opID)
	return true
}

// relayToOthers forwards an op to all connected peers except the source.
func (t *QuicTransport) relayToOthers(sourcePeerID string, op irohengine.WriteOp) {
	data, err := encodeOp(op)
	if err != nil {
		return
	}

	t.mu.RLock()
	var targets []*peerConn
	for id, pc := range t.conns {
		if id != sourcePeerID {
			targets = append(targets, pc)
		}
	}
	t.mu.RUnlock()

	for _, pc := range targets {
		go func(pc *peerConn) {
			if t.cfg.poolStreams {
				t.sendOpPooled(pc, data)
			} else {
				t.sendOp(pc.conn, data)
			}
		}(pc)
	}
}
