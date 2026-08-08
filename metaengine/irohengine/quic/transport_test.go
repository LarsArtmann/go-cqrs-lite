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

// eventuallyGet retries a MapGet until it matches or times out.
func eventuallyGet(
	g gomega.Gomega,
	node metaengine.Engine,
	collection, key string,
	expected any,
	timeout time.Duration,
) {
	g.Eventually(func(g gomega.Gomega) {
		val, ok, err := node.(metaengine.MapBackend).MapGet(context.Background(), collection, key)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(ok).To(gomega.BeTrue())
		g.Expect(val).To(gomega.Equal(expected))
	}, timeout, 50*time.Millisecond).Should(gomega.Succeed())
}

// quicCluster bundles the two engines + transports + gomega + context for a
// single convergence test. Setup cleanup is wired via t.Cleanup.
type quicCluster struct {
	G     gomega.Gomega
	Ctx   context.Context //nolint:containedctx // test helper
	NodeA metaengine.Engine
	NodeB metaengine.Engine
	TA    *quic.QuicTransport
	TB    *quic.QuicTransport
}

// setupTwoNodeQuic creates two connected QuicTransport nodes with engines.
func setupTwoNodeQuic(t *testing.T) (
	nodeA, nodeB metaengine.Engine,
	tA, tB *quic.QuicTransport,
) {
	t.Helper()
	g := gomega.NewWithT(t)

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	tB, err = quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	nodeA = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)
	return nodeA, nodeB, tA, tB
}

// newQuicCluster is the per-test preamble: gomega + context + 2-node setup
// with cleanup deferred via t.Cleanup. Every test in this file uses the same
// preamble; the helper keeps each test focused on its ADT-specific assertions.
func newQuicCluster(t *testing.T) *quicCluster {
	t.Helper()

	nodeA, nodeB, tA, tB := setupTwoNodeQuic(t)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	return &quicCluster{
		G:     gomega.NewWithT(t),
		Ctx:   context.Background(),
		NodeA: nodeA,
		NodeB: nodeB,
		TA:    tA,
		TB:    tB,
	}
}

func TestQuicMapConvergence2Node(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "users", "u1",
		map[string]any{"name": "Alice"})).To(gomega.Succeed())

	eventuallyGet(c.G, c.NodeB, "users", "u1",
		map[string]any{"name": "Alice"}, 5*time.Second)

	t.Logf("node A ID: %s", c.TA.NodeID())
	t.Logf("node B ID: %s", c.TB.NodeID())
}

func TestQuicMapConvergenceBidirectional(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "orders", "o1", "pending")).
		To(gomega.Succeed())
	eventuallyGet(c.G, c.NodeB, "orders", "o1", "pending", 5*time.Second)

	c.G.Expect(c.NodeB.(metaengine.MapBackend).MapSet(c.Ctx, "orders", "o2", "shipped")).
		To(gomega.Succeed())
	eventuallyGet(c.G, c.NodeA, "orders", "o2", "shipped", 5*time.Second)
}

func TestQuicPNCounter(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.CounterBackend).CounterIncrement(c.Ctx, "visits",
		metaengine.Delta{"total": 5})).To(gomega.Succeed())
	c.G.Expect(c.NodeB.(metaengine.CounterBackend).CounterIncrement(c.Ctx, "visits",
		metaengine.Delta{"total": 3})).To(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		counts, err := c.NodeA.(metaengine.CounterBackend).CounterGet(c.Ctx, "visits")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(counts["total"]).To(gomega.Equal(int64(8)),
			"PN-counter should sum both increments")
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		countsB, err := c.NodeB.(metaengine.CounterBackend).CounterGet(c.Ctx, "visits")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(countsB["total"]).To(gomega.Equal(int64(8)))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

func TestQuicSetConvergence(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.SetBackend).SetAdd(c.Ctx, "tags", "go")).To(gomega.Succeed())
	c.G.Expect(c.NodeA.(metaengine.SetBackend).SetAdd(c.Ctx, "tags", "cqrs")).To(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		sb := c.NodeB.(metaengine.SetBackend)
		containsGo, err := sb.SetContains(c.Ctx, "tags", "go")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(containsGo).To(gomega.BeTrue())
		containsCqrs, err := sb.SetContains(c.Ctx, "tags", "cqrs")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(containsCqrs).To(gomega.BeTrue())
	}, 15*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

func TestQuicLWWResolution(t *testing.T) {
	c := newQuicCluster(t)

	// NodeA writes first. Wait for replication to NodeB before the second
	// write. This guarantees wall-clock ordering (T2 > T1) without an
	// arbitrary sleep: both nodes share the same system clock, and the
	// replication round-trip consumes real time.
	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "users", "u1", "Alice-old")).
		To(gomega.Succeed())
	eventuallyGet(c.G, c.NodeB, "users", "u1", "Alice-old", 5*time.Second)

	c.G.Expect(c.NodeB.(metaengine.MapBackend).MapSet(c.Ctx, "users", "u1", "Bob-new")).
		To(gomega.Succeed())

	eventuallyGet(c.G, c.NodeA, "users", "u1", "Bob-new", 5*time.Second)
	eventuallyGet(c.G, c.NodeB, "users", "u1", "Bob-new", 5*time.Second)
}

func TestQuicRTTMeasurement(t *testing.T) {
	c := newQuicCluster(t)

	for range 10 {
		c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "kvs",
			"key", "value")).To(gomega.Succeed())
	}

	profile := c.NodeA.Profile()
	t.Logf("Profile: ReplicationLag=%s NetworkRTT=%s", profile.ReplicationLag, profile.NetworkRTT)
	c.G.Expect(profile.Replication).To(gomega.Equal(metaengine.ReplicationLeaderless))
}

func TestQuicLogConvergence(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.LogBackend).LogAppend(c.Ctx, "audit", "user-login")).
		To(gomega.Succeed())
	c.G.Expect(c.NodeA.(metaengine.LogBackend).LogAppend(c.Ctx, "audit", "file-upload")).
		To(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		entries, err := c.NodeB.(metaengine.LogBackend).LogTail(c.Ctx, "audit", 10)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(entries).To(gomega.HaveLen(2))
		g.Expect(entries[0]).To(gomega.Equal("user-login"))
		g.Expect(entries[1]).To(gomega.Equal("file-upload"))
	}, 30*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

// TestQuicMapUpdateDoesNotReplicate proves that MapUpdate (atomic
// read-modify-write) executes locally but does NOT cross node boundaries over
// QUIC. This is the CALM theorem constraint: non-monotonic operations cannot
// converge via CRDT. Mirrors the in-process TestMapUpdateDoesNotReplicate but
// over the real QUIC transport.
func TestQuicMapUpdateDoesNotReplicate(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "counters", "c1", 0)).
		To(gomega.Succeed())

	// Wait for MapSet to replicate
	c.G.Eventually(func(g gomega.Gomega) {
		_, ok, err := c.NodeB.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(ok).To(gomega.BeTrue())
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	c.G.Expect(c.NodeA.(metaengine.MapUpdater).MapUpdate(c.Ctx, "counters", "c1",
		func(prev any) any {
			n, _ := prev.(int)
			return n + 1
		})).To(gomega.Succeed())

	// Local MapUpdate applied
	c.G.Eventually(func(g gomega.Gomega) {
		valA, _, err := c.NodeA.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(valA).To(gomega.Equal(1))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	time.Sleep(500 * time.Millisecond) // give replication time if it (incorrectly) happened

	valB, okB, err := c.NodeB.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
	c.G.Expect(err).NotTo(gomega.HaveOccurred())
	c.G.Expect(okB).To(gomega.BeTrue(), "MapSet replicated")
	c.G.Expect(valB).To(gomega.Equal(0),
		"MapUpdate must NOT replicate over QUIC (non-CRDT)")
}

func TestQuicMultimapConvergence(t *testing.T) {
	c := newQuicCluster(t)

	mmbA := c.NodeA.(metaengine.MultimapBackend)
	mmbB := c.NodeB.(metaengine.MultimapBackend)

	c.G.Expect(mmbA.MultiAdd(c.Ctx, "members", "team-a", "alice")).To(gomega.Succeed())
	c.G.Expect(mmbA.MultiAdd(c.Ctx, "members", "team-a", "bob")).To(gomega.Succeed())
	c.G.Expect(mmbB.MultiAdd(c.Ctx, "members", "team-a", "carol")).To(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		vals, err := mmbA.MultiGet(c.Ctx, "members", "team-a")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(vals).To(gomega.ConsistOf("alice", "bob", "carol"))
	}, 10*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	c.G.Eventually(func(g gomega.Gomega) {
		vals, err := mmbB.MultiGet(c.Ctx, "members", "team-a")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(vals).To(gomega.ConsistOf("alice", "bob", "carol"))
	}, 10*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}
