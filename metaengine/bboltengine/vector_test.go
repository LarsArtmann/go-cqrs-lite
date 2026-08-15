package bboltengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewBboltVectorEngine(tb testing.TB) metaengine.VectorBackend {
	tb.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		tb.Skipf("bbolt not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng.(metaengine.VectorBackend)
}

func TestBboltVector_InsertAndSearchCosine(t *testing.T) {
	t.Parallel()

	vb := mustNewBboltVectorEngine(t)
	ctx := context.Background()

	embeddings := []metaengine.Embedding{
		{ID: "v1", Values: []float32{1, 0, 0}},
		{ID: "v2", Values: []float32{0.9, 0.1, 0}},
		{ID: "v3", Values: []float32{0, 1, 0}},
	}
	for _, emb := range embeddings {
		if err := vb.VectorInsert(ctx, "vec_bolt", emb); err != nil {
			t.Fatalf("VectorInsert %s: %v", emb.ID, err)
		}
	}

	results, err := vb.VectorSearch(ctx, "vec_bolt", []float32{1, 0, 0}, 2, "cosine")
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

func TestBboltVector_MetricsAndParityWithMemory(t *testing.T) {
	t.Parallel()

	vb := mustNewBboltVectorEngine(t)
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
		if err := vb.VectorInsert(ctx, "vec_bolt_metrics", emb); err != nil {
			t.Fatalf("VectorInsert: %v", err)
		}

		if err := mem.Insert(ctx, "vec_bolt_metrics", emb); err != nil {
			t.Fatalf("memory Insert: %v", err)
		}
	}

	for _, metric := range []string{"cosine", "dot", "euclidean", ""} {
		got, err := vb.VectorSearch(ctx, "vec_bolt_metrics", query, 0, metric)
		if err != nil {
			t.Fatalf("VectorSearch(%q): %v", metric, err)
		}

		want, err := mem.Search(ctx, "vec_bolt_metrics", query, 0, metric)
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

func TestBboltVector_UpsertOverwrites(t *testing.T) {
	t.Parallel()

	vb := mustNewBboltVectorEngine(t)
	ctx := context.Background()

	col := "vec_bolt_upsert"

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{ID: "x", Values: []float32{1, 0}}); err != nil {
		t.Fatalf("VectorInsert 1: %v", err)
	}

	if err := vb.VectorInsert(ctx, col, metaengine.Embedding{ID: "x", Values: []float32{0, 1}}); err != nil {
		t.Fatalf("VectorInsert 2: %v", err)
	}

	results, err := vb.VectorSearch(ctx, col, []float32{0, 1}, 1, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (upsert, no duplicate), got %d: %v", len(results), results)
	}
}

func TestBboltVector_ProfileDeclaresDegradedVector(t *testing.T) {
	t.Parallel()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}
	defer func() { _ = eng.Close() }()

	profile := eng.Profile()

	c, ok := profile.Supports[metaengine.ADTVector]
	if !ok {
		t.Fatal("bbolt profile should declare ADTVector in Supports")
	}

	if c != metaengine.ComplexityON {
		t.Errorf("ADTVector complexity = %s, want %s (brute-force)", c, metaengine.ComplexityON)
	}

	if !profile.IsDegraded(metaengine.ADTVector) {
		t.Error("ADTVector should be declared degraded on bbolt (no ANN index)")
	}
}
