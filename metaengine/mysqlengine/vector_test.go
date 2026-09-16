package mysqlengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMySQLVectorRoundtrip exercises the brute-force VectorBackend on a live
// MySQL/MariaDB server: insert, k-NN ordering per metric, exact distance
// parity with metaengine.VectorDistance, filtered k-NN, counter, and reset.
// NOT parallel: it ends with a total ResetEngine over the shared persistent
// database (see TestResetEngine_ClearsEveryADT).
func TestMySQLVectorRoundtrip(t *testing.T) {
	if mysqlTestDSN() == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL integration test")
	}

	eng := mustNewMySQLEngine(t)

	//art-dupl:accept dep-isolated dialect twin (dgraphengine/duckdbengine vector_test.go)
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

	if results[0].Distance != metaengine.VectorDistance(query, []float32{1, 0}, "cosine") {
		t.Fatalf(
			"top distance = %v, want %v",
			results[0].Distance,
			metaengine.VectorDistance(query, []float32{1, 0}, "cosine"),
		)
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

	if err := eng.(interface {
		ResetEngine(ctx context.Context) error
	}).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if n, err := vc.VectorCount(ctx, col); err != nil || n != 0 {
		t.Fatalf("VectorCount after reset = %d err=%v, want 0", n, err)
	}
}
