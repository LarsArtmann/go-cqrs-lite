//go:build cgo

package quic

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestEvictPooledStream_ReopenOnNextSend pins the self-healing half of the
// pooling contract (pool.go): a stream error evicts the pooled BiStream, and
// the NEXT sendOpPooled opens a fresh one. The error itself is simulated by
// calling evictPooledStream directly — the same single code path every
// frame-write/ack failure funnels through — because injecting a fault into the
// FFI stream is not possible from outside the process.
func TestEvictPooledStream_ReopenOnNextSend(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	receiver, err := New(WithLocalOnly(), WithRelay(), WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer receiver.Close()

	nodeRecv := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("evict-receiver"),
		irohengine.WithTransport(receiver),
	)
	defer nodeRecv.Close()

	ticket, err := receiver.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	sender, err := New(WithLocalOnly(), WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer sender.Close()

	nodeSend := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("evict-sender"),
		irohengine.WithTransport(sender),
	)
	defer nodeSend.Close()

	g.Expect(sender.Connect(ticket)).To(gomega.Succeed())
	g.Eventually(func() int { return sender.PeerCount() }, 5*time.Second, 50*time.Millisecond).
		Should(gomega.Equal(1))

	mb := nodeSend.(metaengine.MapBackend)

	// Op 1 opens the pooled stream.
	g.Expect(mb.MapSet(ctx, "evict", "k1", "before")).To(gomega.Succeed())

	var pc *peerConn
	g.Eventually(func(g gomega.Gomega) {
		peerID := receiver.NodeID()

		sender.mu.RLock()
		pc = sender.conns[peerID]
		sender.mu.RUnlock()

		g.Expect(pc).NotTo(gomega.BeNil())
		g.Expect(pc.stream).NotTo(gomega.BeNil())
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	// Simulate the error path: eviction must clear the stream ...
	sender.evictPooledStream(pc)

	pc.streamMu.Lock()
	cleared := pc.stream == nil
	pc.streamMu.Unlock()

	if !cleared {
		t.Fatal("evictPooledStream left pc.stream non-nil")
	}

	// ... and the next send must reopen a fresh stream and still deliver.
	g.Expect(mb.MapSet(ctx, "evict", "k2", "after")).To(gomega.Succeed())
	g.Eventually(func(g gomega.Gomega) {
		got, ok, err := nodeRecv.(metaengine.MapBackend).MapGet(ctx, "evict", "k2")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(ok).To(gomega.BeTrue())
		g.Expect(got).To(gomega.Equal("after"))
	}, 10*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	if opened := sender.StreamsOpenedForPeer(receiver.NodeID()); opened != 2 {
		t.Errorf("StreamsOpenedForPeer = %d, want 2 (one before evict, one after)", opened)
	}
}
