package irohengine

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ClusterFactory creates a 2-node replicated cluster for convergence testing.
// The factory wires transports, connects peers, waits for connection readiness,
// and registers cleanup via t.Cleanup. The caller only needs the two engines.
//
// Each transport (in-process, loopback, QUIC) provides its own factory
// implementation and calls RunConvergenceSuite once.
type ClusterFactory func(t *testing.T) (nodeA, nodeB metaengine.Engine)

// RunConvergenceSuite runs the standard convergence test battery against a
// 2-node cluster created by factory. This eliminates ~200 lines of duplicated
// convergence tests across the in-process, loopback, and QUIC transport test
// files.
//
// The suite covers the 8 common CRDT convergence scenarios:
//   - MapConvergence (A→B MapSet/MapGet)
//   - Bidirectional (A→B and B→A)
//   - CounterConvergence (PN-Counter)
//   - SetConvergence (OR-Set)
//   - LogConvergence (append-only log)
//   - MultimapConvergence (OR-Set per key)
//   - GraphConvergence (per-edge LWW: adds replicate)
//   - GraphEdgeRemovalConvergence (per-edge LWW: removes replicate)
//
// Transport-specific tests (LWW with clock, RTT measurement, protocol mismatch,
// stream pooling) remain in their respective test files. Polling helpers live
// in convergence_poll.go.
func RunConvergenceSuite(t *testing.T, factory ClusterFactory) {
	t.Helper()

	t.Run("MapConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		expected := map[string]any{"name": "Alice"}
		mustNoErr(t, nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", expected))
		waitForMap(t, nodeB, "users", "u1", expected)
	})

	t.Run("Bidirectional", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.MapBackend).MapSet(ctx, "orders", "o1", "pending"))
		waitForMap(t, nodeB, "orders", "o1", "pending")

		mustNoErr(t, nodeB.(metaengine.MapBackend).MapSet(ctx, "orders", "o2", "shipped"))
		waitForMap(t, nodeA, "orders", "o2", "shipped")
	})

	t.Run("CounterConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.CounterBackend).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": 5}))
		mustNoErr(t, nodeB.(metaengine.CounterBackend).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": 3}))

		waitForCounter(t, nodeA, "visits", "total", 8)
		waitForCounter(t, nodeB, "visits", "total", 8)
	})

	t.Run("SetConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "go"))
		mustNoErr(t, nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "cqrs"))

		waitForSetContains(t, nodeB, "tags", "go")
		waitForSetContains(t, nodeB, "tags", "cqrs")
	})

	t.Run("LogConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "user-login"))
		mustNoErr(t, nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "file-upload"))

		waitForLogTail(t, nodeB, "audit", []string{"user-login", "file-upload"})
	})

	t.Run("MultimapConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mmbA := nodeA.(metaengine.MultimapBackend)
		mmbB := nodeB.(metaengine.MultimapBackend)

		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", "alice"))
		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", "bob"))
		mustNoErr(t, mmbB.MultiAdd(ctx, "members", "team-a", "carol"))

		waitForMultimap(t, nodeA, "members", "team-a", []string{"alice", "bob", "carol"})
		waitForMultimap(t, nodeB, "members", "team-a", []string{"alice", "bob", "carol"})
	})

	t.Run("GraphConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		gd := nodeA.(graphDispatch)
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"}))
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "carol"}))
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "bob", To: "dave"}))

		waitForGraphNeighbors(t, nodeB, "follows", "alice", 1, []string{"bob", "carol"})
		waitForGraphNeighbors(t, nodeB, "follows", "alice", 2, []string{"bob", "carol", "dave"})
	})

	t.Run("GraphEdgeRemovalConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		gd := nodeA.(graphDispatch)
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"}))
		waitForGraphNeighbors(t, nodeB, "follows", "alice", 1, []string{"bob"})

		mustNoErr(t, nodeA.(graphRemoveDispatch).
			GraphRemoveEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"}))
		waitForGraphNeighbors(t, nodeB, "follows", "alice", 1, []string{})
	})
}
