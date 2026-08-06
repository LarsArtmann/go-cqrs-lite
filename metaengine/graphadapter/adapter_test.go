package graphadapter_test

import (
	"context"
	"testing"

	graphadapter "github.com/larsartmann/go-cqrs-lite/metaengine/graphadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestAdapter_Profile(t *testing.T) {
	t.Parallel()

	a := graphadapter.New()
	defer a.Close()

	p := a.Profile()
	if p.Name != "graph-memory" {
		t.Errorf("Name = %q, want %q", p.Name, "graph-memory")
	}
	if p.Supports[metaengine.ADTGraph] != metaengine.ComplexityON {
		t.Error("expected ADTGraph at O(N)")
	}
}

func TestAdapter_GraphAddEdgeAndNeighbors(t *testing.T) {
	t.Parallel()

	a := graphadapter.New()
	defer a.Close()

	ctx := context.Background()

	if err := a.GraphAddEdge(ctx, "assign", metaengine.Edge{From: "alice", To: "t1"}); err != nil {
		t.Fatalf("GraphAddEdge 1: %v", err)
	}
	if err := a.GraphAddEdge(ctx, "assign", metaengine.Edge{From: "alice", To: "t2"}); err != nil {
		t.Fatalf("GraphAddEdge 2: %v", err)
	}
	if err := a.GraphAddEdge(ctx, "assign", metaengine.Edge{From: "alice", To: "t3"}); err != nil {
		t.Fatalf("GraphAddEdge 3: %v", err)
	}

	neighbors, err := a.GraphNeighbors(ctx, "assign", "alice", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}
	if len(neighbors) != 3 {
		t.Fatalf("expected 3 neighbors, got %d", len(neighbors))
	}
}

func TestAdapter_ImplementsInterfaces(t *testing.T) {
	t.Parallel()

	var eng metaengine.Engine = graphadapter.New()
	defer eng.Close()

	var gb metaengine.GraphBackend = eng.(metaengine.GraphBackend)
	if gb == nil {
		t.Fatal("Adapter does not implement GraphBackend")
	}
}
