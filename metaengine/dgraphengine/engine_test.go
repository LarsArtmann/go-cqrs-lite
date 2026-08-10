package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphBackend is the local graph dispatch contract for tests. The Dgraph
// engine implements this structurally via its GraphAddEdge/GraphNeighbors
// methods (ADR-0113: the exported metaengine.GraphBackend was deleted).
type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// TestProfile verifies the Dgraph engine profile without requiring a running
// Dgraph instance. The profile values are compile-time constants.
func TestProfile(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	profile := eng.Profile()

	if profile.Name != "dgraph" {
		t.Errorf("profile.Name = %q, want %q", profile.Name, "dgraph")
	}

	if profile.Persistence != metaengine.PersistencePersistent {
		t.Errorf("profile.Persistence = %q, want %q",
			profile.Persistence, metaengine.PersistencePersistent)
	}

	if profile.Replication != metaengine.ReplicationSingleLeader {
		t.Errorf("profile.Replication = %q, want %q",
			profile.Replication, metaengine.ReplicationSingleLeader)
	}

	// Graph should be supported and NOT degraded — Dgraph's native strength.
	complexity, ok := profile.Supports[metaengine.ADTGraph]
	if !ok {
		t.Fatal("profile does not support ADTGraph")
	}

	if complexity != metaengine.ComplexityODegree {
		t.Errorf("ADTGraph complexity = %q, want %q",
			complexity, metaengine.ComplexityODegree)
	}

	if profile.IsDegraded(metaengine.ADTGraph) {
		t.Error("ADTGraph should not be degraded — Dgraph's native strength")
	}

	// Map should be supported.
	if _, ok := profile.Supports[metaengine.ADTMap]; !ok {
		t.Error("profile does not support ADTMap")
	}

	// Search should be supported and NOT degraded — term index is native.
	if _, ok := profile.Supports[metaengine.ADTSearch]; !ok {
		t.Error("profile does not support ADTSearch")
	}

	if profile.IsDegraded(metaengine.ADTSearch) {
		t.Error("ADTSearch should not be degraded — term index is native")
	}
}

// TestMapBackend exercises the Map ADT (MapSet, MapGet, MapDelete).
func TestMapBackend(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	// Set and get.
	if err := mb.MapSet(ctx, "test-map", "k1", map[string]any{"v": 1}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	val, ok, err := mb.MapGet(ctx, "test-map", "k1")
	if err != nil {
		t.Fatalf("MapGet: %v", err)
	}

	if !ok {
		t.Fatal("MapGet: expected ok=true")
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("MapGet: expected map[string]any, got %T", val)
	}

	if m["v"] != float64(1) {
		t.Errorf("MapGet: value = %v, want 1", m["v"])
	}

	// Missing key.
	_, ok, err = mb.MapGet(ctx, "test-map", "nonexistent")
	if err != nil {
		t.Fatalf("MapGet missing: %v", err)
	}

	if ok {
		t.Error("MapGet missing: expected ok=false")
	}

	// Delete.
	if err := mb.MapDelete(ctx, "test-map", "k1"); err != nil {
		t.Fatalf("MapDelete: %v", err)
	}

	_, ok, err = mb.MapGet(ctx, "test-map", "k1")
	if err != nil {
		t.Fatalf("MapGet after delete: %v", err)
	}

	if ok {
		t.Error("MapGet after delete: expected ok=false")
	}
}

// TestGraphBackend exercises the Graph ADT — Dgraph's killer feature.
func TestGraphBackend(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	ctx := context.Background()
	gb := eng.(graphBackend)

	// Build: A→B, A→C, B→D (bidirectional).
	edges := []metaengine.Edge{
		{From: "X", To: "Y"},
		{From: "X", To: "Z"},
		{From: "Y", To: "W"},
	}

	for _, e := range edges {
		if err := gb.GraphAddEdge(ctx, "graph-test", e); err != nil {
			t.Fatalf("GraphAddEdge %v: %v", e, err)
		}
	}

	// Neighbors of X at depth 1 should be Y and Z.
	neighbors, err := gb.GraphNeighbors(ctx, "graph-test", "X", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	seen := make(map[string]bool)
	for _, n := range neighbors {
		seen[fmt.Sprint(n)] = true
	}

	if !seen["Y"] {
		t.Error("GraphNeighbors: expected Y in results")
	}

	if !seen["Z"] {
		t.Error("GraphNeighbors: expected Z in results")
	}

	if seen["W"] {
		t.Error("GraphNeighbors: W should not be reachable at depth 1")
	}

	if seen["X"] {
		t.Error("GraphNeighbors: start node X should not be in results")
	}
}
