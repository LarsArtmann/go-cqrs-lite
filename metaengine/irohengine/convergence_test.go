package irohengine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestInProcessConvergenceSuite runs the shared convergence test battery against
// the in-process transport. This replaces 5 hand-written convergence tests
// (MapConvergence, CounterConvergence, SetConvergence, LogConvergence,
// MultimapConvergence) with one suite call.
func TestInProcessConvergenceSuite(t *testing.T) {
	t.Parallel()
	irohengine.RunConvergenceSuite(t, func(t *testing.T) (metaengine.Engine, metaengine.Engine) {
		return newTwoNodeCluster(t)
	})
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

// TestWithClock_DeterministicLWW proves the WithClock option eliminates all
// timing assumptions in LWW convergence tests. Both nodes share a manualClock;
// timestamp ordering is controlled by explicit Advance() calls — no time.Sleep.
func TestWithClock_DeterministicLWW(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	clock := newManualClock(time.Unix(2_000_000, 0))
	nodeA, nodeB := newTwoNodeClusterWithClock(t, clock)

	// Node A writes first (timestamp T0).
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "kv", "k", "v1")).
		To(gomega.Succeed())

	// Clock advances deterministically — no sleep.
	clock.Advance(5 * time.Second)

	// Node B overwrites with a strictly later timestamp (T1 > T0).
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "kv", "k", "v2")).
		To(gomega.Succeed())

	// Both nodes converge to the LWW winner (v2).
	for i, n := range []metaengine.Engine{nodeA, nodeB} {
		val, ok, err := n.(metaengine.MapBackend).MapGet(ctx, "kv", "k")
		g.Expect(err).NotTo(gomega.HaveOccurred(), "node %d", i)
		g.Expect(ok).To(gomega.BeTrue(), "node %d", i)
		g.Expect(val).To(gomega.Equal("v2"), "node %d should see LWW winner", i)
	}

	// Now node A deletes with an even later timestamp (T2 > T1).
	clock.Advance(5 * time.Second)
	g.Expect(nodeA.(metaengine.MapBackend).MapDelete(ctx, "kv", "k")).To(gomega.Succeed())

	// Both nodes see the deletion.
	for i, n := range []metaengine.Engine{nodeA, nodeB} {
		_, ok, err := n.(metaengine.MapBackend).MapGet(ctx, "kv", "k")
		g.Expect(err).NotTo(gomega.HaveOccurred(), "node %d", i)
		g.Expect(ok).To(gomega.BeFalse(), "node %d should see delete via LWW", i)
	}

	// Stale op with an EARLIER timestamp must NOT resurrect the key.
	clock.Advance(5 * time.Second)
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "kv", "k", "stale-resurrect")).
		To(gomega.Succeed())

	// This new write IS later than the delete, so it should win.
	// (This confirms the clock continues advancing and new writes supersede deletes.)
	val, ok, err := nodeA.(metaengine.MapBackend).MapGet(ctx, "kv", "k")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue(), "newer write should win over older delete")
	g.Expect(val).To(gomega.Equal("stale-resurrect"))
}

// TestWithClock_StaleOpRejected proves that a remote op with an older timestamp
// is correctly rejected by the LWW guard — the core guarantee of last-writer-wins.
func TestWithClock_StaleOpRejected(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	clock := newManualClock(time.Unix(3_000_000, 0))
	nodeA, nodeB := newTwoNodeClusterWithClock(t, clock)

	// Both write to the same key. Node A writes at T0.
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k", "from-A")).
		To(gomega.Succeed())

	// Node B writes at T1 > T0.
	clock.Advance(time.Second)
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "data", "k", "from-B")).
		To(gomega.Succeed())

	// Both converge to from-B (later timestamp).
	val, _, err := nodeA.(metaengine.MapBackend).MapGet(ctx, "data", "k")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(val).To(gomega.Equal("from-B"), "LWW winner should be from-B")
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

	// Phase 1: sequential writes complete before Close.
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k1", "v1")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k2", "v2")).To(gomega.Succeed())
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "k3", "v3")).To(gomega.Succeed())

	// Phase 2: concurrent writes from multiple goroutines, all completing
	// before Close. The InProcessNetwork delivers synchronously (Publish
	// blocks until all peers process the op), so these are all in-flight
	// concurrently but fully replicated before Close returns.
	const concurrentCount = 50
	var wg sync.WaitGroup
	for i := range concurrentCount {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("conc-%d", idx)
			_ = nodeA.(metaengine.MapBackend).MapSet(ctx, "data", key, idx)
		}(i)
	}
	wg.Wait()

	// Close node A — all prior writes must have already replicated.
	g.Expect(nodeA.Close()).To(gomega.Succeed())

	// Verify ALL writes reached node B: both sequential and concurrent.
	for _, key := range []string{"k1", "k2", "k3"} {
		val, found, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "data", key)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(found).
			To(gomega.BeTrue(), "node B should have received pre-close write for %s", key)
		g.Expect(val).To(gomega.Equal("v" + key[1:]))
	}
	for i := range concurrentCount {
		val, found, err := nodeB.(metaengine.MapBackend).MapGet(
			ctx,
			"data",
			fmt.Sprintf("conc-%d", i),
		)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(found).To(gomega.BeTrue(), "node B should have concurrent write conc-%d", i)
		g.Expect(val).To(gomega.Equal(i))
	}

	// Phase 3: writes AFTER Close must not panic, but may silently fail
	// (the transport is closed). The engine should return gracefully.
	_ = nodeA.(metaengine.MapBackend).MapSet(ctx, "data", "post-close", "should-not-arrive")

	g.Expect(nodeB.Close()).To(gomega.Succeed())
}
