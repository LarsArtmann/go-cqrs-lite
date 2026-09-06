package irohengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphDispatcher mirrors metaengine's unexported graph dispatch contract
// (ADR-0113). Tests assert the wrapper forwards it structurally — graphadapter
// and the capability audit rely on that detection, not on Profile() claims.
type graphDispatcher interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// TestReplicatedForwardsGraphDispatch pins the wrapper-capability-preservation
// contract: Profile() copies the local engine's graph declaration, so the
// wrapper MUST also satisfy the structural graph dispatch contract. This
// regressed silently before explicit forwarding (conformance: OVER-DECLARED).
func TestReplicatedForwardsGraphDispatch(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := irohengine.Replicated(metaengine.NewMemoryEngine())
	defer eng.Close()

	g.Expect(metaengine.HasGraphSupport(eng)).To(gomega.BeTrue())

	gd, ok := eng.(graphDispatcher)
	g.Expect(ok).To(gomega.BeTrue())

	ctx := context.Background()
	g.Expect(gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "a", To: "b"})).
		To(gomega.Succeed())
	g.Expect(gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "a", To: "c"})).
		To(gomega.Succeed())
	g.Expect(gd.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "b", To: "d"})).
		To(gomega.Succeed())

	neighbors, err := gd.GraphNeighbors(ctx, "follows", "a", 1)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(neighbors).To(gomega.ConsistOf("b", "c"))

	neighbors, err = gd.GraphNeighbors(ctx, "follows", "a", 2)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(neighbors).To(gomega.ConsistOf("b", "c", "d"))
}

// graphlessLocal is a minimal Engine without any graph methods.
type graphlessLocal struct{}

func (graphlessLocal) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "graphless"}
}

func (graphlessLocal) Close() error { return nil }

// TestReplicatedGraph_GraphlessLocalErrors verifies the forwarded methods
// degrade honestly when the wrapped engine lacks graph support instead of
// panicking on a failed type assertion.
func TestReplicatedGraph_GraphlessLocalErrors(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := irohengine.Replicated(graphlessLocal{})
	defer eng.Close()

	gd, ok := eng.(graphDispatcher)
	g.Expect(ok).To(gomega.BeTrue())

	err := gd.GraphAddEdge(context.Background(), "follows", metaengine.Edge{From: "a", To: "b"})
	g.Expect(err).To(gomega.MatchError(irohengine.ErrGraphBackendNotImplemented))

	neighbors, err := gd.GraphNeighbors(context.Background(), "follows", "a", 1)
	g.Expect(err).To(gomega.MatchError(irohengine.ErrGraphBackendNotImplemented))
	g.Expect(neighbors).To(gomega.BeNil())
}

// graphEdgeRemover mirrors the optional edge-removal extension for tests.
type graphEdgeRemover interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
}

// TestGraphEdgeConvergenceCrossPeer pins the core graph-WriteOp convergence
// contract: edges added through the wrapper on node A replicate to node B and
// back (bidirectional), and a remove converges the edge away on the peer.
// Before the OpGraphAddEdge/OpGraphRemoveEdge wire kinds (2026-09-06), edges
// were local-only passthrough and these peer assertions could never pass.
func TestGraphEdgeConvergenceCrossPeer(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB := newTwoNodeCluster(t)

	a := nodeA.(graphDispatcher)
	b := nodeB.(graphDispatcher)

	g.Expect(a.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"})).
		To(gomega.Succeed())
	neighbors, err := b.GraphNeighbors(ctx, "follows", "alice", 1)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(neighbors).To(gomega.ConsistOf("bob"))

	g.Expect(b.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "carol", To: "alice"})).
		To(gomega.Succeed())
	neighbors, err = a.GraphNeighbors(ctx, "follows", "carol", 1)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(neighbors).To(gomega.ConsistOf("alice"))

	g.Expect(nodeA.(graphEdgeRemover).
		GraphRemoveEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"})).
		To(gomega.Succeed())
	neighbors, err = b.GraphNeighbors(ctx, "follows", "alice", 1)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(neighbors).To(gomega.BeEmpty(), "removed edge must converge away on the peer")
}

// TestGraphEdgeLWW_StaleAddDoesNotResurrect pins per-edge LWW register
// semantics: a remove with a newer timestamp wins everywhere, and a later add
// carrying an OLDER timestamp (a lagging node's reordered write) must be
// rejected by every peer that saw the remove — no resurrection. Node C runs
// on a clock set 10s behind A/B, so its writes are deterministically stale.
func TestGraphEdgeLWW_StaleAddDoesNotResurrect(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	mainClock := newManualClock(time.Unix(4_000_000, 0))
	staleClock := newManualClock(time.Unix(4_000_000, 0).Add(-10 * time.Second))

	net := irohengine.NewNetwork()
	join := func(id string, clock *manualClock) metaengine.Engine {
		eng := irohengine.Replicated(
			metaengine.NewMemoryEngine(),
			irohengine.WithAuthor("node-"+id),
			irohengine.WithTransport(net.Join(id)),
			irohengine.WithClock(clock),
		)
		t.Cleanup(func() { _ = eng.Close() })
		return eng
	}
	nodeA := join("a", mainClock)
	nodeB := join("b", mainClock)
	nodeC := join("c", staleClock)

	// Stale node C adds x→y: nothing newer exists yet, so A and B apply it.
	g.Expect(nodeC.(graphDispatcher).
		GraphAddEdge(ctx, "follows", metaengine.Edge{From: "x", To: "y"})).
		To(gomega.Succeed())
	for _, node := range []metaengine.Engine{nodeA, nodeB} {
		neighbors, err := node.(graphDispatcher).GraphNeighbors(ctx, "follows", "x", 1)
		g.Expect(err).To(gomega.Succeed())
		g.Expect(neighbors).To(gomega.ConsistOf("y"))
	}

	// B removes at a timestamp strictly newer than everything C has minted.
	g.Expect(nodeB.(graphEdgeRemover).
		GraphRemoveEdge(ctx, "follows", metaengine.Edge{From: "x", To: "y"})).
		To(gomega.Succeed())
	for _, node := range []metaengine.Engine{nodeA, nodeC} {
		neighbors, err := node.(graphDispatcher).GraphNeighbors(ctx, "follows", "x", 1)
		g.Expect(err).To(gomega.Succeed())
		g.Expect(neighbors).To(gomega.BeEmpty(), "remove must converge to every peer")
	}

	// C re-adds the SAME edge x→y, still on its lagging clock: the op's
	// timestamp is older than B's remove, so A and B must reject it — the
	// removed edge must not resurrect via a stale write. (A re-add of a
	// DIFFERENT edge is unaffected: per-edge LWW is keyed by (from, to).)
	g.Expect(nodeC.(graphDispatcher).
		GraphAddEdge(ctx, "follows", metaengine.Edge{From: "x", To: "y"})).
		To(gomega.Succeed())
	for _, node := range []metaengine.Engine{nodeA, nodeB} {
		neighbors, err := node.(graphDispatcher).GraphNeighbors(ctx, "follows", "x", 1)
		g.Expect(err).To(gomega.Succeed())
		g.Expect(neighbors).To(gomega.BeEmpty(), "stale add must not resurrect removed adjacency")
	}
}
