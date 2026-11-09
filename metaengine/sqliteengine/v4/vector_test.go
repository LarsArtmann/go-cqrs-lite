package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func newVectorEngine(t *testing.T) (metaengine.Engine, context.Context) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	return eng, context.Background()
}

func insertTestVectors(
	t *testing.T,
	eng metaengine.Engine,
	ctx context.Context,
	col string,
) {
	t.Helper()

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
}

func TestVectorSearch_MetricsOrderCorrect(t *testing.T) {
	eng, ctx := newVectorEngine(t)
	col := t.Name()

	insertTestVectors(t, eng, ctx, col)

	vb := eng.(metaengine.VectorBackend)

	cases := []struct {
		metric string
		want   []string
	}{
		{metric: "cosine", want: []string{"a", "c", "b"}},
		{metric: "euclidean", want: []string{"a", "c", "b"}},
		{metric: "dot", want: []string{"a", "c", "b"}},
	}

	for _, tc := range cases {
		results, err := vb.VectorSearch(ctx, col, []float32{1, 0}, 3, tc.metric)
		if err != nil {
			t.Fatalf("VectorSearch(%s): %v", tc.metric, err)
		}

		if len(results) != len(tc.want) {
			t.Fatalf(
				"VectorSearch(%s): got %d results, want %d",
				tc.metric,
				len(results),
				len(tc.want),
			)
		}

		for i, wantID := range tc.want {
			if results[i].ID != wantID {
				t.Fatalf("VectorSearch(%s)[%d] = %s (d=%v), want %s",
					tc.metric, i, results[i].ID, results[i].Distance, wantID)
			}
		}
	}
}

func TestVectorSearch_ParityWithReferenceDistance(t *testing.T) {
	eng, ctx := newVectorEngine(t)
	col := t.Name()

	insertTestVectors(t, eng, ctx, col)

	query := []float32{0.8, 0.2}

	results, err := eng.(metaengine.VectorBackend).VectorSearch(ctx, col, query, 2, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	reference := metaengine.VectorDistance(query, []float32{0.9, 0.1}, "cosine")
	if results[0].ID != "c" || results[0].Distance != reference {
		t.Fatalf("top result = %+v, want id c with distance %v", results[0], reference)
	}
}

func TestVectorUpsert_ReplacesMetadata(t *testing.T) {
	eng, ctx := newVectorEngine(t)
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

	results, err := fb.VectorSearchFiltered(ctx, col, []float32{0, 1}, 1, "euclidean",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "x"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("stale metadata survived upsert: %+v", results)
	}

	plain, err := vb.VectorSearch(ctx, col, []float32{0, 1}, 1, "euclidean")
	if err != nil || len(plain) != 1 || plain[0].ID != "a" {
		t.Fatalf("reinserted vector not found: %+v err=%v", plain, err)
	}
}

func TestVectorSearchFiltered_FiltersBeforeRanking(t *testing.T) {
	eng, ctx := newVectorEngine(t)
	col := t.Name()

	insertTestVectors(t, eng, ctx, col)

	results, err := eng.(metaengine.VectorFilterBackend).VectorSearchFiltered(ctx, col,
		[]float32{1, 0}, 1, "cosine",
		[]metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "y"}})
	if err != nil {
		t.Fatalf("VectorSearchFiltered: %v", err)
	}

	if len(results) != 1 || results[0].ID != "b" {
		t.Fatalf("filtered k-NN = %+v, want exactly b", results)
	}
}

func TestVectorCounter_CountsAndLists(t *testing.T) {
	eng, ctx := newVectorEngine(t)

	col := t.Name()

	insertTestVectors(t, eng, ctx, col)

	vc := eng.(metaengine.VectorCounter)

	n, err := vc.VectorCount(ctx, col)
	if err != nil || n != 3 {
		t.Fatalf("VectorCount = %d err=%v, want 3", n, err)
	}

	if n, err := vc.VectorCount(ctx, "missing"); err != nil || n != 0 {
		t.Fatalf("VectorCount(missing) = %d err=%v, want 0", n, err)
	}

	cols, err := vc.VectorCollections(ctx)
	if err != nil || len(cols) != 1 || cols[0] != col {
		t.Fatalf("VectorCollections = %v err=%v, want [%s]", cols, err, col)
	}
}

func TestVectorReset_ClearsEmbeddings(t *testing.T) {
	eng, ctx := newVectorEngine(t)
	col := t.Name()

	insertTestVectors(t, eng, ctx, col)

	if err := eng.(interface {
		ResetEngine(ctx context.Context) error
	}).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	n, err := eng.(metaengine.VectorCounter).VectorCount(ctx, col)
	if err != nil || n != 0 {
		t.Fatalf("VectorCount after reset = %d err=%v, want 0", n, err)
	}
}
