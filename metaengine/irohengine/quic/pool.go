//go:build cgo

package quic

import (
	"log/slog"
	"time"

	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"
)

// sendOpPooled writes an encoded op to a persistent BiStream using length-prefix
// framing, then reads a minimal ack. The persistent stream is opened lazily on
// first use and reused for all subsequent ops to the same peer.
//
// On any stream error, the pooled stream is evicted so the next call reopens
// a fresh stream — self-healing after connection resets or transient failures.
func (t *QuicTransport) sendOpPooled(pc *peerConn, data []byte) {
	pc.streamMu.Lock()
	defer pc.streamMu.Unlock()

	if pc.stream == nil {
		stream, err := pc.conn.OpenBi()
		if err != nil {
			slog.Warn("quic sendOpPooled: OpenBi failed",
				slog.String("peer", pc.peerID), slog.Any("err", err))
			return
		}
		pc.stream = stream
	}

	// Write frame: [4-byte length][payload]
	if err := pc.stream.Send().WriteAll(frameHeader(len(data))); err != nil {
		t.evictPooledStream(pc)
		return
	}
	if err := pc.stream.Send().WriteAll(data); err != nil {
		t.evictPooledStream(pc)
		return
	}

	// Read ack frame: [4-byte length][payload]
	ackHeader, err := pc.stream.Recv().ReadExact(frameHeaderSize)
	if err != nil {
		t.evictPooledStream(pc)
		return
	}
	ackSize, err := parseFrameHeader(ackHeader)
	if err != nil {
		t.evictPooledStream(pc)
		return
	}
	if ackSize > 0 {
		if _, err := pc.stream.Recv().ReadExact(ackSize); err != nil {
			t.evictPooledStream(pc)
			return
		}
	}

	// Record real RTT from QUIC's ACK timing
	if rtt := pc.conn.Rtt(); rtt != nil {
		t.recordRTT(time.Duration(*rtt))
	}
}

// evictPooledStream closes and nils the pooled stream so the next sendOpPooled
// call opens a fresh one. Called under pc.streamMu.
func (t *QuicTransport) evictPooledStream(pc *peerConn) {
	if pc.stream != nil {
		_ = pc.stream.Send().Finish()
		pc.stream.Destroy()
		pc.stream = nil
	}
}

// handlePooledStream reads length-prefixed frames from a persistent BiStream
// in a loop, dispatching each op to local subscribers. After processing each
// op, it writes a minimal ack frame back to the sender.
//
// This is the receive-side counterpart to sendOpPooled. One persistent stream
// handles an unlimited number of ops — no per-op stream creation overhead.
func (t *QuicTransport) handlePooledStream(
	conn *iroh_ffi.Connection,
	sourcePeerID string,
	stream *iroh_ffi.BiStream,
) {
	for {
		// Read frame header
		header, err := stream.Recv().ReadExact(frameHeaderSize)
		if err != nil {
			return // stream closed
		}
		size, err := parseFrameHeader(header)
		if err != nil {
			return
		}

		// Read frame payload
		data, err := stream.Recv().ReadExact(size)
		if err != nil {
			return
		}

		// Send ack: [4-byte length=1][1-byte payload]
		_ = stream.Send().WriteAll(frameHeader(1))
		_ = stream.Send().WriteAll([]byte{1})

		// Decode and dispatch
		op, err := decodeOp(data)
		if err != nil {
			continue
		}

		// Op-level dedup
		if !t.markSeen(op.ID) {
			continue
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
}
