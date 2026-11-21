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

// Suite fixture identities and counter deltas: 5 + 3 converges to 8 on both
// nodes — the PN-Counter invariant under test.
const (
	suitePeerAlice = "alice"
	suitePeerBob   = "bob"
	suitePeerCarol = "carol"
	suitePeerDave  = "dave"
	suiteDeltaA    = 5
	suiteDeltaB    = 3
	suiteDeltaSum  = 8
)

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
		mustNoErr(t, asMapBackend(t, nodeA).MapSet(ctx, "users", "u1", expected))
		waitForMap(t, nodeB, "users", "u1", expected)
	})

	t.Run("Bidirectional", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, asMapBackend(t, nodeA).MapSet(ctx, "orders", "o1", "pending"))
		waitForMap(t, nodeB, "orders", "o1", "pending")

		mustNoErr(t, asMapBackend(t, nodeB).MapSet(ctx, "orders", "o2", "shipped"))
		waitForMap(t, nodeA, "orders", "o2", "shipped")
	})

	t.Run("CounterConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, asCounterBackend(t, nodeA).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": suiteDeltaA}))
		mustNoErr(t, asCounterBackend(t, nodeB).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": suiteDeltaB}))

		waitForCounter(t, nodeA, "visits", "total", suiteDeltaSum)
		waitForCounter(t, nodeB, "visits", "total", suiteDeltaSum)
	})

	t.Run("SetConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, asSetBackend(t, nodeA).SetAdd(ctx, "tags", "go"))
		mustNoErr(t, asSetBackend(t, nodeA).SetAdd(ctx, "tags", "cqrs"))

		waitForSetContains(t, nodeB, "tags", "go")
		waitForSetContains(t, nodeB, "tags", "cqrs")
	})

	t.Run("LogConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, asLogBackend(t, nodeA).LogAppend(ctx, "audit", "user-login"))
		mustNoErr(t, asLogBackend(t, nodeA).LogAppend(ctx, "audit", "file-upload"))

		waitForLogTail(t, nodeB, "audit", []string{"user-login", "file-upload"})
	})

	t.Run("MultimapConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mmbA := asMultimapBackend(t, nodeA)
		mmbB := asMultimapBackend(t, nodeB)

		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", suitePeerAlice))
		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", suitePeerBob))
		mustNoErr(t, mmbB.MultiAdd(ctx, "members", "team-a", suitePeerCarol))

		waitForMultimap(t, nodeA, "members", "team-a", []string{suitePeerAlice, suitePeerBob, suitePeerCarol})
		waitForMultimap(t, nodeB, "members", "team-a", []string{suitePeerAlice, suitePeerBob, suitePeerCarol})
	})

	t.Run("GraphConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		gd := asGraphCapable(t, nodeA)
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: suitePeerAlice, To: suitePeerBob}))
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: suitePeerAlice, To: suitePeerCarol}))
		mustNoErr(t, gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: suitePeerBob, To: suitePeerDave}))

		waitForGraphNeighbors(t, nodeB, "follows", suitePeerAlice, 1, []string{suitePeerBob, suitePeerCarol})
		waitForGraphNeighbors(
			t,
			nodeB,
			"follows",
			suitePeerAlice,
			2,
			[]string{suitePeerBob, suitePeerCarol, suitePeerDave},
		)
	})

	t.Run("GraphEdgeRemovalConvergence", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		nodeA, nodeB := factory(t)

		gd := asGraphCapable(t, nodeA)
		mustNoErr(t, gd.GraphAddEdge(ctx, "blocks", metaengine.Edge{From: "mallory", To: suitePeerAlice}))
		waitForGraphNeighbors(t, nodeB, "blocks", "mallory", 1, []string{suitePeerAlice})

		remover, ok := nodeA.(graphRemoveDispatch)
		if !ok {
			t.Fatalf("engine %T does not implement the edge-removal dispatch contract", nodeA)
		}

		mustNoErr(t, remover.
			GraphRemoveEdge(ctx, "blocks", metaengine.Edge{From: "mallory", To: suitePeerAlice}))
		waitForGraphNeighbors(t, nodeB, "blocks", "mallory", 1, []string{})
	})
}
