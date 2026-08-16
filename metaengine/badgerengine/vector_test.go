package badgerengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend (parity with the pebble/bbolt precedent) ---

func TestBadgerVector_InsertAndSearchCosine(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorBackend)
	ctx := context.Background()

	embeddings := []metaengine.Embedding{
		{ID: "v1", Values: []float32{1, 0, 0}},
		{ID: "v2", Values: []float32{0.9, 0.1, 0}},
		{ID: "v3", Values: []float32{0, 1, 0}},
	}
	for _, emb := range embeddings {
		if err := vb.VectorInsert(ctx, "vec_badger", emb); err != nil {
			t.Fatalf("VectorInsert %s: %v", emb.ID, err)
		}
	}

	results, err := vb.VectorSearch(ctx, "vec_badger", []float32{1, 0, 0}, 2, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected k=2 results, got %d: %v", len(results), results)
	}

	if results[0].ID != "v1" {
		t.Errorf("nearest = %s (dist %f), want v1", results[0].ID, results[0].Distance)
	}

	if results[1].ID != "v2" {
		t.Errorf("second = %s, want v2 (closest angular neighbor)", results[1].ID)
	}
}

func TestBadgerVector_MetricsAndParityWithMemory(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorBackend)
	ctx := context.Background()

	query := []float32{1, 2, 3}
	mem := metaengine.NewMemoryVectorIndex()

	for i, values := range [][]float32{
		{1, 0, 0},
		{0.5, 0.5, 0.5},
		{3, 2, 1},
		{-1, -2, -3},
	} {
		emb := metaengine.Embedding{ID: string(rune('a' + i)), Values: values}
		if err := vb.VectorInsert(ctx, "vec_badger_metrics", emb); err != nil {
			t.Fatalf("VectorInsert: %v", err)
		}

		if err := mem.Insert(ctx, "vec_badger_metrics", emb); err != nil {
			t.Fatalf("memory Insert: %v", err)
		}
	}

	for _, metric := range []string{"cosine", "dot", "euclidean", ""} {
		got, err := vb.VectorSearch(ctx, "vec_badger_metrics", query, 0, metric)
		if err != nil {
			t.Fatalf("VectorSearch(%q): %v", metric, err)
		}

		want, err := mem.Search(ctx, "vec_badger_metrics", query, 0, metric)
		if err != nil {
			t.Fatalf("memory Search(%q): %v", metric, err)
		}

		if len(got) != len(want) {
			t.Fatalf("metric %q: got %d results, memory got %d", metric, len(got), len(want))
		}

		for i := range got {
			if got[i].ID != want[i].ID {
				t.Errorf("metric %q: rank %d = %s, memory ranks it %s",
					metric, i, got[i].ID, want[i].ID)
			}
		}
	}
}

func TestBadgerVector_UpsertOverwrites(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorBackend)
	ctx := context.Background()

	col := "vec_badger_upsert"

	for _, values := range [][]float32{{1, 0}, {0, 1}} {
		if err := vb.VectorInsert(
			ctx,
			col,
			metaengine.Embedding{ID: "x", Values: values},
		); err != nil {
			t.Fatalf("VectorInsert: %v", err)
		}
	}

	results, err := vb.VectorSearch(ctx, col, []float32{0, 1}, 1, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (upsert, no duplicate), got %d: %v", len(results), results)
	}
}

func TestBadgerVector_EmptyCollection(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorBackend)

	results, err := vb.VectorSearch(context.Background(), "vec_badger_empty",
		[]float32{1, 0}, 5, "euclidean")
	if err != nil {
		t.Fatalf("VectorSearch on empty collection: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results, got %v", results)
	}
}

// --- VectorFilterBackend (metadata-filtered k-NN) ---

func TestBadgerVector_FilteredSearch(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorFilterBackend)
	ctx := context.Background()

	embeddings := []metaengine.Embedding{
		{ID: "near-a", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "a"}},
		{ID: "near-b", Values: []float32{0.95, 0.05}, Metadata: map[string]any{"tenant": "b"}},
		{ID: "far-a", Values: []float32{0, 1}, Metadata: map[string]any{"tenant": "a"}},
	}
	for _, emb := range embeddings {
		if err := vb.VectorInsert(ctx, "vec_badger_filtered", emb); err != nil {
			t.Fatalf("VectorInsert %s: %v", emb.ID, err)
		}
	}

	// Tenant-filtered top-2 must be the two NEAREST tenant-a vectors, not the
	// global top-2 with tenant-b matches dropped.
	results, err := vb.VectorSearchFiltered(ctx, "vec_badger_filtered",
		[]float32{1, 0}, 2, "cosine",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "a"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 filtered results, got %d: %v", len(results), results)
	}

	if results[0].ID != "near-a" || results[1].ID != "far-a" {
		t.Errorf("filtered k-NN = [%s, %s], want [near-a, far-a]",
			results[0].ID, results[1].ID)
	}
}

func TestBadgerVector_FilteredSearchUpsertClearsStaleMetadata(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorFilterBackend)
	ctx := context.Background()

	col := "vec_badger_meta_upsert"

	insert := func(meta map[string]any) {
		t.Helper()
		emb := metaengine.Embedding{ID: "x", Values: []float32{1, 0}, Metadata: meta}
		if err := vb.VectorInsert(ctx, col, emb); err != nil {
			t.Fatalf("VectorInsert: %v", err)
		}
	}

	insert(map[string]any{"tenant": "a"})
	insert(nil) // upsert without metadata must clear the stale map

	results, err := vb.VectorSearchFiltered(ctx, col, []float32{1, 0}, 5, "cosine",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "a"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected stale metadata to be cleared (0 matches), got %v", results)
	}
}

func TestBadgerVector_CollectionsAreIsolated(t *testing.T) {
	t.Parallel()

	vb := mustNewBadgerEngine(t).(metaengine.VectorBackend)
	ctx := context.Background()

	for _, col := range []string{"vec_badger_c1", "vec_badger_c2"} {
		emb := metaengine.Embedding{ID: "same-id", Values: []float32{1, 0}}
		if err := vb.VectorInsert(ctx, col, emb); err != nil {
			t.Fatalf("VectorInsert %s: %v", col, err)
		}
	}

	for _, col := range []string{"vec_badger_c1", "vec_badger_c2"} {
		results, err := vb.VectorSearch(ctx, col, []float32{1, 0}, 10, "euclidean")
		if err != nil {
			t.Fatalf("VectorSearch %s: %v", col, err)
		}

		if len(results) != 1 {
			t.Errorf("collection %s: expected 1 isolated entry, got %d", col, len(results))
		}
	}
}

func TestBadgerEngine_ImplementsVectorAndGraphBackends(t *testing.T) {
	t.Parallel()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		t.Skipf("Badger not available: %v", err)
	}
	defer func() { _ = eng.Close() }()

	if _, ok := eng.(metaengine.VectorBackend); !ok {
		t.Error("badger engine should implement VectorBackend (pebble/bbolt parity)")
	}

	if _, ok := eng.(metaengine.VectorFilterBackend); !ok {
		t.Error("badger engine should implement VectorFilterBackend")
	}

	if !metaengine.HasGraphSupport(eng) {
		t.Error("badger engine should implement graphBackend (GraphAddEdge + GraphNeighbors)")
	}

	if !metaengine.HasGraphEdgeRemoval(eng) {
		t.Error("badger engine should implement GraphRemoveEdge")
	}

	if !metaengine.HasUndirectedGraphSupport(eng) {
		t.Error("badger engine should implement GraphNeighborsUndirected")
	}

	p := eng.Profile()
	if p.Supports[metaengine.ADTVector] != metaengine.ComplexityON ||
		!p.DegradedADTs[metaengine.ADTVector] {
		t.Errorf("ADTVector must be declared ComplexityON + degraded, got %v degraded=%v",
			p.Supports[metaengine.ADTVector], p.DegradedADTs[metaengine.ADTVector])
	}

	if p.Supports[metaengine.ADTGraph] != metaengine.ComplexityODegree {
		t.Errorf("ADTGraph must be declared ComplexityODegree, got %v",
			p.Supports[metaengine.ADTGraph])
	}
}
