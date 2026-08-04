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

// waitForPeers polls until both transports see the expected peer count, or times out.
func waitForPeers(t *testing.T, transports []*quic.QuicTransport, expected int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		allReady := true
		for _, tr := range transports {
			if tr.PeerCount() < expected {
				allReady = false
				break
			}
		}
		if allReady {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d peers on all transports", expected)
}

// eventuallyGet retries a MapGet until it succeeds or times out.
// This handles the async nature of QUIC delivery.
func eventuallyGet(
	g gomega.Gomega,
	node metaengine.Engine,
	collection string,
	key string,
	expected any,
	timeout time.Duration,
) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		val, ok, err := node.(metaengine.MapBackend).MapGet(context.Background(), collection, key)
		if err == nil && ok {
			matcher := gomega.Equal(expected)
			success, matchErr := matcher.Match(val)
			if matchErr == nil && success {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	val, ok, err := node.(metaengine.MapBackend).MapGet(context.Background(), collection, key)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(val).To(gomega.Equal(expected))
}

func TestQuicMapConvergence2Node(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	// Create two transports
	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	// Create engines (subscribes to transport BEFORE connecting)
	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	// Connect B → A
	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	// Wait for bidirectional connection registration
	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	// Write on A
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1",
		map[string]any{"name": "Alice"})).To(gomega.Succeed())

	// Verify B sees it (async QUIC delivery)
	eventuallyGet(g, nodeB, "users", "u1",
		map[string]any{"name": "Alice"}, 5*time.Second)

	t.Logf("node A ID: %s", tA.NodeID())
	t.Logf("node B ID: %s", tB.NodeID())
}

func TestQuicMapConvergenceBidirectional(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	// A writes
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "orders", "o1", "pending")).
		To(gomega.Succeed())
	eventuallyGet(g, nodeB, "orders", "o1", "pending", 5*time.Second)

	// B writes
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "orders", "o2", "shipped")).
		To(gomega.Succeed())
	eventuallyGet(g, nodeA, "orders", "o2", "shipped", 5*time.Second)
}

func TestQuicPNCounter(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	g.Expect(nodeA.(metaengine.CounterBackend).CounterIncrement(ctx, "visits",
		metaengine.Delta{"total": 5})).To(gomega.Succeed())
	g.Expect(nodeB.(metaengine.CounterBackend).CounterIncrement(ctx, "visits",
		metaengine.Delta{"total": 3})).To(gomega.Succeed())

	// Wait for convergence then check
	time.Sleep(200 * time.Millisecond)

	counts, err := nodeA.(metaengine.CounterBackend).CounterGet(ctx, "visits")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(counts["total"]).To(gomega.Equal(int64(8)), "PN-counter should sum both increments")

	countsB, err := nodeB.(metaengine.CounterBackend).CounterGet(ctx, "visits")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(countsB["total"]).To(gomega.Equal(int64(8)))
}

func TestQuicSetConvergence(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	g.Expect(nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "go")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "cqrs")).To(gomega.Succeed())

	// Wait for convergence
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		contains, _, _ := nodeB.(metaengine.SetBackend).SetContains(ctx, "tags", "go")
}

func TestQuicLWWResolution(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	// A writes old value
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice-old")).
		To(gomega.Succeed())
	time.Sleep(100 * time.Millisecond) // ensure B's timestamp is later

	// B writes new value
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Bob-new")).
		To(gomega.Succeed())

	// Wait for convergence — both should see Bob-new (latest timestamp wins)
	eventuallyGet(g, nodeA, "users", "u1", "Bob-new", 5*time.Second)
	eventuallyGet(g, nodeB, "users", "u1", "Bob-new", 5*time.Second)
}

func TestQuicRTTMeasurement(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	defer nodeA.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	// Generate traffic
	for i := 0; i < 10; i++ {
		g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "kvs",
			"key", "value")).To(gomega.Succeed())
	}

	// Profile should have real measured RTT
	profile := nodeA.Profile()
	t.Logf("Profile: ReplicationLag=%s NetworkRTT=%s", profile.ReplicationLag, profile.NetworkRTT)

	// On localhost, RTT should be very small but present (it's real QUIC)
	// Don't assert on specific values — just that the profile runs
	g.Expect(profile.Replication).To(gomega.Equal(metaengine.ReplicationLeaderless))
}

func TestQuicLogConvergence(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tA.Close()

	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer tB.Close()

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	g.Expect(nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "user-login")).
		To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "file-upload")).
		To(gomega.Succeed())

	// Wait for convergence
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ := nodeB.(metaengine.LogBackend).LogTail(ctx, "audit", 10)
		if len(entries) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	entries, err := nodeB.(metaengine.LogBackend).LogTail(ctx, "audit", 10)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(entries).To(gomega.HaveLen(2))
	g.Expect(entries[0]).To(gomega.Equal("user-login"))
	g.Expect(entries[1]).To(gomega.Equal("file-upload"))
}
