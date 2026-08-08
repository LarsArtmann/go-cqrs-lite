package irohengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestMapConvergence2Node(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	nodeA, nodeB := newTwoNodeCluster(t)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", map[string]any{"name": "Alice"})).
		To(gomega.Succeed())

	val, ok, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(val).To(gomega.Equal(map[string]any{"name": "Alice"}))
}

func TestMapConvergence3Node(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	net := irohengine.NewNetwork()
	nodes := make([]metaengine.Engine, 3)
	for i, id := range []string{"a", "b", "c"} {
		nodes[i] = irohengine.Replicated(
			metaengine.NewMemoryEngine(),
			irohengine.WithAuthor("node-"+id),
			irohengine.WithTransport(net.Join(id)),
		)
		defer nodes[i].Close()
	}

	g.Expect(nodes[0].(metaengine.MapBackend).MapSet(ctx, "orders", "o1", "pending")).
		To(gomega.Succeed())

	for i, n := range nodes {
		val, ok, err := n.(metaengine.MapBackend).MapGet(ctx, "orders", "o1")
		g.Expect(err).NotTo(gomega.HaveOccurred(), "node %d", i)
		g.Expect(ok).To(gomega.BeTrue(), "node %d should see the value", i)
		g.Expect(val).To(gomega.Equal("pending"), "node %d", i)
	}
}

func TestLWWResolution(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	clock := newManualClock(time.Unix(1_000_000, 0))
	nodeA, nodeB := newTwoNodeClusterWithClock(t, clock)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice-old")).
		To(gomega.Succeed())

	// Deterministic timestamp advance — no time.Sleep needed.
	// Both nodes share the same clock, so node B's write gets a strictly
	// later timestamp, guaranteeing LWW resolution.
	clock.Advance(time.Second)

	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Bob-new")).
		To(gomega.Succeed())

	valA, _, err := nodeA.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(valA).To(gomega.Equal("Bob-new"), "node A should have latest value via LWW")

	valB, _, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(valB).To(gomega.Equal("Bob-new"))
}

func TestPNCounter(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	nodeA, nodeB := newTwoNodeCluster(t)

	g.Expect(nodeA.(metaengine.CounterBackend).CounterIncrement(ctx, "visits", metaengine.Delta{"total": 5})).
		To(gomega.Succeed())
	g.Expect(nodeB.(metaengine.CounterBackend).CounterIncrement(ctx, "visits", metaengine.Delta{"total": 3})).
		To(gomega.Succeed())

	counts, err := nodeA.(metaengine.CounterBackend).CounterGet(ctx, "visits")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(counts["total"]).To(gomega.Equal(int64(8)), "PN-counter should sum both increments")

	countsB, err := nodeB.(metaengine.CounterBackend).CounterGet(ctx, "visits")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(countsB["total"]).To(gomega.Equal(int64(8)))
}

func TestSetConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	nodeA, nodeB := newTwoNodeCluster(t)

	g.Expect(nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "go")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "cqrs")).To(gomega.Succeed())

	contains, err := nodeB.(metaengine.SetBackend).SetContains(ctx, "tags", "go")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(contains).To(gomega.BeTrue())

	contains, err = nodeB.(metaengine.SetBackend).SetContains(ctx, "tags", "cqrs")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(contains).To(gomega.BeTrue())
}

func TestLogConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	nodeA, nodeB := newTwoNodeCluster(t)

	g.Expect(nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "user-login")).
		To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "file-upload")).
		To(gomega.Succeed())

	entries, err := nodeB.(metaengine.LogBackend).LogTail(ctx, "audit", 10)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(entries).To(gomega.HaveLen(2))
	g.Expect(entries[0]).To(gomega.Equal("user-login"))
	g.Expect(entries[1]).To(gomega.Equal("file-upload"))
}

func TestMultimapConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	nodeA, nodeB := newTwoNodeCluster(t)

	g.Expect(nodeA.(metaengine.MultimapBackend).MultiAdd(ctx, "members", "team-a", "alice")).
		To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.MultimapBackend).MultiAdd(ctx, "members", "team-a", "bob")).
		To(gomega.Succeed())
	g.Expect(nodeB.(metaengine.MultimapBackend).MultiAdd(ctx, "members", "team-a", "carol")).
		To(gomega.Succeed())

	vals, err := nodeA.(metaengine.MultimapBackend).MultiGet(ctx, "members", "team-a")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(vals).To(gomega.ConsistOf("alice", "bob", "carol"))
}

func TestMapDeleteLWWConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	clock := newManualClock(time.Unix(1_000_000, 0))
	nodeA, nodeB := newTwoNodeClusterWithClock(t, clock)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice")).
		To(gomega.Succeed())

	valB, found, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(valB).To(gomega.Equal("Alice"))

	// Deterministic: delete gets a strictly later timestamp than the set,
	// so the delete wins LWW on node A — no time.Sleep needed.
	clock.Advance(time.Second)
	g.Expect(nodeB.(metaengine.MapBackend).MapDelete(ctx, "users", "u1")).To(gomega.Succeed())

	_, foundA, err := nodeA.(metaengine.MapBackend).MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundA).To(gomega.BeFalse(), "node A should see deletion via LWW convergence")
}

func TestGracefulShutdown_InflightOps(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	net := irohengine.NewNetwork()
	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(net.Join("a")),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(net.Join("b")),
	)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k1", "v1")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k2", "v2")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k3", "v3")).To(gomega.Succeed())

	g.Expect(nodeA.Close()).To(gomega.Succeed())
	time.Sleep(20 * time.Millisecond)

	for _, key := range []string{"k1", "k2", "k3"} {
		val, found, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "data", key)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(found).
			To(gomega.BeTrue(), "node B should have received pre-close write for %s", key)
		g.Expect(val).To(gomega.Equal("v" + key[1:]))
	}

	g.Expect(nodeB.Close()).To(gomega.Succeed())
}
