package tursoengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestVectorSearch_LibSQLPushdown exercises the libSQL vector-distance
// path: the delegated sqliteengine probes vector32()/vector_distance_* at
// construction, and the turso driver (embedded libSQL) provides them — so
// k-NN scoring happens in SQL, not in Go. Ordering and distances must match
// metaengine.VectorDistance semantics exactly.
func TestVectorSearch_LibSQLPushdown(t *testing.T) {
	eng, err := tursoengine.New("")
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	col := t.Name()

	vb := eng.(metaengine.VectorBackend)

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

	for _, metric := range []string{"cosine", "euclidean", "dot"} {
		results, err := vb.VectorSearch(ctx, col, []float32{1, 0}, 3, metric)
		if err != nil {
			t.Fatalf("VectorSearch(%s): %v", metric, err)
		}

		want := []string{"a", "c", "b"}
		for i, wantID := range want {
			if results[i].ID != wantID {
				t.Fatalf("VectorSearch(%s)[%d] = %s, want %s", metric, i, results[i].ID, wantID)
			}
		}

		reference := metaengine.VectorDistance([]float32{1, 0}, []float32{1, 0}, metric)
		if results[0].Distance != reference {
			t.Fatalf("VectorSearch(%s) top distance = %v, want %v",
				metric, results[0].Distance, reference)
		}
	}
}

// TestVectorSearchFiltered_LibSQL asserts filtered k-NN AND semantics on the
// delegated engine (filters evaluate in Go, scoring shared with memory).
func TestVectorSearchFiltered_LibSQL(t *testing.T) {
	eng, err := tursoengine.New("")
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	col := t.Name()

	vb := eng.(metaengine.VectorFilterBackend)

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{
		ID: "a", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "x"},
	}); err != nil {
		t.Fatalf("insert a: %v", err)
	}

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{
		ID: "b", Values: []float32{0, 1}, Metadata: map[string]any{"tenant": "y"},
	}); err != nil {
		t.Fatalf("insert b: %v", err)
	}

	results, err := vb.VectorSearchFiltered(ctx, col, []float32{1, 0}, 1, "cosine",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "y"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(results) != 1 || results[0].ID != "b" {
		t.Fatalf("filtered k-NN = %+v, want exactly b", results)
	}
}
