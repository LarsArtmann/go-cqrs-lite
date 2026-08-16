package graphadapter_test

import (
	"context"
	"testing"

	graphadapter "github.com/larsartmann/go-cqrs-lite/metaengine/graphadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// graphadapter implements only GraphBackend (not MapBackend), so
// RunRecordStampTest does not apply — AutoInsert requires MapBackend to
// store projected values. All 7 MapBackend-capable engines (Memory, Pebble,
// SQLite, DuckDB, Postgres, Badger, Dgraph) have record-stamp coverage.
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

	if !metaengine.HasGraphSupport(eng) {
		t.Fatal("Adapter does not implement graph dispatch (GraphAddEdge + GraphNeighbors)")
	}

	if !metaengine.HasGraphEdgeRemoval(eng) {
		t.Fatal("Adapter should implement GraphRemoveEdge (driver RemoveEdge)")
	}

	// Undirected traversal is deliberately NOT implemented: the driver's
	// Traverse walks outgoing edges only, so undirected queries must fail
	// with the explicit missing-capability error instead of wrong nodes.
	if metaengine.HasUndirectedGraphSupport(eng) {
		t.Fatal("Adapter must NOT implement GraphNeighborsUndirected (outgoing-only driver)")
	}
}

func TestAdapter_GraphRemoveEdge(t *testing.T) {
	t.Parallel()

	a := graphadapter.New()
	defer a.Close()

	ctx := context.Background()

	for _, e := range []metaengine.Edge{
		{From: "alice", To: "t1"},
		{From: "alice", To: "t2"},
	} {
		if err := a.GraphAddEdge(ctx, "assign", e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	if err := a.GraphRemoveEdge(
		ctx,
		"assign",
		metaengine.Edge{From: "alice", To: "t1"},
	); err != nil {
		t.Fatalf("GraphRemoveEdge: %v", err)
	}

	neighbors, err := a.GraphNeighbors(ctx, "assign", "alice", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors after removal: %v", err)
	}

	if len(neighbors) != 1 || neighbors[0] != "t2" {
		t.Errorf("after removal = %v, want [t2]", neighbors)
	}

	// Idempotent: removing a missing edge is a no-op.
	if err := a.GraphRemoveEdge(
		ctx,
		"assign",
		metaengine.Edge{From: "alice", To: "t1"},
	); err != nil {
		t.Fatalf("GraphRemoveEdge (re-remove): %v", err)
	}
}

func TestAdapter_StoreIntegration_RecordAware(t *testing.T) {
	t.Parallel()

	type TaskAssigned struct {
		UserID string
		TaskID string
	}

	type AssignmentsInput struct {
		UserID string
		Depth  int
	}

	eng := graphadapter.New()
	defer eng.Close()

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[AssignmentsInput, string](
			"assignments",
			metaengine.OnRecord(
				TaskAssigned{},
				func(_ record.Record, e TaskAssigned) metaengine.Edge {
					return metaengine.Edge{From: e.UserID, To: e.TaskID}
				},
			),
		),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()

	assignments := []TaskAssigned{
		{UserID: "alice", TaskID: "t1"},
		{UserID: "alice", TaskID: "t2"},
		{UserID: "alice", TaskID: "t3"},
	}

	for i, a := range assignments {
		rec := record.Record{
			Type:       "TaskAssigned",
			StreamID:   record.NewStreamRef("User", a.UserID),
			StreamType: "User",
			Version:    int64(i + 1),
		}
		if err := store.ApplyRecord(ctx, rec, a); err != nil {
			t.Fatalf("ApplyRecord %d: %v", i, err)
		}
	}

	result, err := store.Execute(AssignmentsInput{UserID: "alice", Depth: 1})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	neighbors, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(neighbors) != 3 {
		t.Errorf("expected 3 neighbors, got %d: %v", len(neighbors), neighbors)
	}
}
