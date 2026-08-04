//go:build cgo

package quic_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestQuic3NodeRelayConvergence(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	// Coordinator with relay enabled
	coord, err := quic.New(quic.WithLocalOnly(), quic.WithRelay())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer coord.Close()

	nodeCoord := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("coordinator"),
		irohengine.WithTransport(coord),
	)
	defer nodeCoord.Close()

	ticket, err := coord.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Node A connects to coordinator
	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	defer nodeA.Close()

	g.Expect(tA.Connect(ticket)).To(gomega.Succeed())

	// Node B connects to coordinator
	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeB.Close()

	g.Expect(tB.Connect(ticket)).To(gomega.Succeed())

	// Wait for all connections
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if coord.PeerCount() >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	g.Expect(coord.PeerCount()).To(gomega.BeNumerically(">=", 2),
		"coordinator should have 2 peers")

	time.Sleep(500 * time.Millisecond)

	// Coordinator writes
	g.Expect(nodeCoord.(metaengine.MapBackend).MapSet(ctx, "test", "k1", "from-coord")).
		To(gomega.Succeed())

	// Both nodes should eventually see it (via coordinator relay or direct)
	eventuallyGet(g, nodeA, "test", "k1", "from-coord", 10*time.Second)
	eventuallyGet(g, nodeB, "test", "k1", "from-coord", 10*time.Second)
}

func TestQuicWriteAfterReconnect(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	// Create coordinator
	coord, err := quic.New(quic.WithLocalOnly(), quic.WithRelay())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer coord.Close()

	ticket, err := coord.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// First node connects
	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	g.Expect(tA.Connect(ticket)).To(gomega.Succeed())
	time.Sleep(500 * time.Millisecond)

	// Write before disconnect
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "persist", "k1", "before")).
		To(gomega.Succeed())
	time.Sleep(500 * time.Millisecond)

	// Close node A
	tA.Close()

	// Coordinator writes while A is offline
	nodeCoord := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("coordinator"),
		irohengine.WithTransport(coord),
	)
	defer nodeCoord.Close()

	g.Expect(nodeCoord.(metaengine.MapBackend).MapSet(ctx, "persist", "k2", "while-offline")).
		To(gomega.Succeed())

	// Node A reconnects with a new transport
	tA2, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA2.Close()

	nodeA2 := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a-2"),
		irohengine.WithTransport(tA2),
	)
	defer nodeA2.Close()

	g.Expect(tA2.Connect(ticket)).To(gomega.Succeed())
	time.Sleep(1 * time.Second)

	// Write after reconnect
	g.Expect(nodeA2.(metaengine.MapBackend).MapSet(ctx, "persist", "k3", "after-reconnect")).
		To(gomega.Succeed())

	// Wait for convergence
	time.Sleep(2 * time.Second)

	// Coordinator should see the new write
	eventuallyGet(g, nodeCoord, "persist", "k3", "after-reconnect", 5*time.Second)
}
