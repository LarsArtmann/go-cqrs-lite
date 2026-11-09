//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestVectorSearch_SQLPushdown exercises the engine-scored k-NN path: insert,
// per-metric ordering, distance parity with metaengine.VectorDistance,
// filtered k-NN, counter, and reset.
func TestVectorSearch_SQLPushdown(t *testing.T) {
	eng := mustNewDuckEngine(t)

	ctx := context.Background()
	col := t.Name()

	vb := eng.(metaengine.VectorBackend)
	fb := eng.(metaengine.VectorFilterBackend)
	vc := eng.(metaengine.VectorCounter)

	embeddings := []metaengine.Embedding{
		{ID: "a", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "x"}},
		{ID: "b", Values: []float32{0, 1}, Metadata: map[string]any{"tenant": "y"}},
		{ID: "c", Values: []float32{0.9, 0.1}},
	}

	for _, emb := range embeddings {
		if err := vb.VectorInsert(ctx, col, emb); err != nil {
			t.Fatalf("VectorInsert %s: %v", emb.ID, err)
		}
	}

	query := []float32{1, 0}

	for _, metric := range []string{"cosine", "euclidean", "dot"} {
		results, err := vb.VectorSearch(ctx, col, query, 3, metric)
		if err != nil {
			t.Fatalf("VectorSearch(%s): %v", metric, err)
		}

		want := []string{"a", "c", "b"}
		for i, wantID := range want {
			if results[i].ID != wantID {
				t.Fatalf("VectorSearch(%s)[%d] = %s, want %s", metric, i, results[i].ID, wantID)
			}
		}
	}

	results, err := vb.VectorSearch(ctx, col, query, 1, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	reference := metaengine.VectorDistance(query, []float32{1, 0}, "cosine")
	if results[0].Distance != reference {
		t.Fatalf("top distance = %v, want %v", results[0].Distance, reference)
	}

	filtered, err := fb.VectorSearchFiltered(ctx, col, query, 1, "cosine",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "y"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(filtered) != 1 || filtered[0].ID != "b" {
		t.Fatalf("filtered k-NN = %+v, want exactly b", filtered)
	}

	if n, err := vc.VectorCount(ctx, col); err != nil || n != 3 {
		t.Fatalf("VectorCount = %d err=%v, want 3", n, err)
	}
}

// TestVectorUpsert_ReplacesMetadata pins the full-replace upsert contract.
func TestVectorUpsert_ReplacesMetadata(t *testing.T) {
	eng := mustNewDuckEngine(t)

	ctx := context.Background()
	col := t.Name()

	vb := eng.(metaengine.VectorBackend)
	fb := eng.(metaengine.VectorFilterBackend)

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{
		ID: "a", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "x"},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{
		ID: "a", Values: []float32{0, 1},
	}); err != nil {
		t.Fatalf("reinsert without metadata: %v", err)
	}

	stale, err := fb.VectorSearchFiltered(ctx, col, []float32{0, 1}, 1, "euclidean",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "x"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(stale) != 0 {
		t.Fatalf("stale metadata survived upsert: %+v", stale)
	}

	if n, err := eng.(metaengine.VectorCounter).VectorCount(ctx, col); err != nil || n != 1 {
		t.Fatalf("VectorCount after upsert = %d err=%v, want 1", n, err)
	}
}

// TestVectorReset_ClearsEmbeddings pins the reset contract for meta_vector.
func TestVectorReset_ClearsEmbeddings(t *testing.T) {
	eng := mustNewDuckEngine(t)

	ctx := context.Background()
	col := t.Name()

	if err := eng.(metaengine.VectorBackend).VectorInsert(ctx, col, metaengine.Embedding{
		ID: "a", Values: []float32{1, 0},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := eng.(interface {
		ResetEngine(ctx context.Context) error
	}).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if n, err := eng.(metaengine.VectorCounter).VectorCount(ctx, col); err != nil || n != 0 {
		t.Fatalf("VectorCount after reset = %d err=%v, want 0", n, err)
	}
}
