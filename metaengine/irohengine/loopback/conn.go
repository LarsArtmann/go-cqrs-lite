package loopback

import (
	"encoding/json/v2"
	"math/rand"
	"net"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
)

func (t *LoopbackTransport) acceptLoop() {
	defer t.acceptWG.Done()
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return // listener closed
		}

		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			_ = conn.Close()
			return
		}
		t.conns[conn.RemoteAddr().String()] = conn
		t.mu.Unlock()

		go t.handleConnection(conn)
	}
}

func (t *LoopbackTransport) handleConnection(conn net.Conn) {
	for {
		data, err := readFrame(conn)
		if err != nil {
			return // connection closed or error
		}

		var op irohengine.WriteOp
		if err := json.Unmarshal(data, &op); err != nil {
			continue
		}

		if !t.markSeen(op.ID) {
			continue
		}

		if !op.PublishedAt.IsZero() {
			t.recordLatency(time.Since(op.PublishedAt))
		}

		if t.maxDelay > 0 {
			//nolint:gosec // G404: math/rand is safe for simulated delay
			time.Sleep(time.Duration(rand.Int63n(t.maxDelay.Nanoseconds())))
		}

		t.mu.RLock()
		subs := t.subs
		t.mu.RUnlock()
		for _, s := range subs {
			s(op)
		}
	}
}

func (t *LoopbackTransport) recordLatency(d time.Duration) {
	t.latencyMu.Lock()
	defer t.latencyMu.Unlock()
	t.latencyMs = append(t.latencyMs, d)
	if len(t.latencyMs) > rttWindowSize {
		t.latencyMs = t.latencyMs[len(t.latencyMs)-rttWindowSize:]
	}
}

func (t *LoopbackTransport) markSeen(opID string) bool {
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
