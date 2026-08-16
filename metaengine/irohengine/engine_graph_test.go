package irohengine_test

import (
	"context"
	"testing"

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
