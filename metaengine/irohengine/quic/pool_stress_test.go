//go:build cgo

package quic_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestQuicPooledThousandOps pins the stream-pooling property under sustained
// load: 1,000 replicated ops across a pooled pair must converge AND ride a
// SINGLE BidirectionalStream (StreamsOpenedForPeer == 1). Without pooling each
// op opens its own stream; a regression that silently reopens per op (or per
// error) turns this counter into ~1000.
func TestQuicPooledThousandOps(t *testing.T) {
	if testing.Short() {
		t.Skip("live QUIC pair required")
	}

	g := gomega.NewWithT(t)
	ctx := context.Background()

	receiver, err := quic.New(quic.WithLocalOnly(), quic.WithRelay(), quic.WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer receiver.Close()

	nodeRecv := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("pooled-receiver"),
		irohengine.WithTransport(receiver),
	)
	defer nodeRecv.Close()

	ticket, err := receiver.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	sender, err := quic.New(quic.WithLocalOnly(), quic.WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer sender.Close()

	nodeSend := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("pooled-sender"),
		irohengine.WithTransport(sender),
	)
	defer nodeSend.Close()

	g.Expect(sender.Connect(ticket)).To(gomega.Succeed())
	waitForPeers(t, []*quic.QuicTransport{sender}, 1)

	mb := nodeSend.(metaengine.MapBackend)

	const ops = 1000
	for i := range ops {
		key := fmt.Sprintf("k-%04d", i)
		g.Expect(mb.MapSet(ctx, "pooled-stress", key, i)).To(gomega.Succeed())
	}

	// All 1,000 ops converge at the receiver.
	for i := range ops {
		key := fmt.Sprintf("k-%04d", i)
		eventuallyGet(g, nodeRecv, "pooled-stress", key, i, 10*time.Second)
	}

	// Pooling held: exactly ONE stream served the whole run. Check AFTER the
	// final op converged, so the sender's pooled stream is the one that
	// delivered everything.
	opened := sender.StreamsOpenedForPeer(receiver.NodeID())
	if opened != 1 {
		t.Errorf("StreamsOpenedForPeer = %d after %d ops, want 1 (pooling regressed?)", opened, ops)
	}
}
