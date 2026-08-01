package metaengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestVectorSearch_MemoryBruteForce(t *testing.T) {
	t.Parallel()

	index := metaengine.NewMemoryVectorIndex()
	ctx := context.Background()

	// Insert embeddings
	index.Insert(ctx, "test", metaengine.Embedding{ID: "a", Values: []float32{1, 0, 0}})
	index.Insert(ctx, "test", metaengine.Embedding{ID: "b", Values: []float32{0, 1, 0}})
	index.Insert(ctx, "test", metaengine.Embedding{ID: "c", Values: []float32{1, 1, 0}})

	// Query: closest to {1, 0, 0} should be "a" then "c"
	results, err := index.Search(ctx, "test", []float32{1, 0, 0}, 2, "euclidean")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].ID != "a" {
		t.Errorf("nearest neighbor: got %q, want %q", results[0].ID, "a")
	}
}

func TestVectorSearch_CosineMetric(t *testing.T) {
	t.Parallel()

	index := metaengine.NewMemoryVectorIndex()
	ctx := context.Background()

	index.Insert(ctx, "test", metaengine.Embedding{ID: "doc1", Values: []float32{0.9, 0.1, 0}})
	index.Insert(ctx, "test", metaengine.Embedding{ID: "doc2", Values: []float32{0.1, 0.9, 0}})

	results, err := index.Search(ctx, "test", []float32{1, 0, 0}, 1, "cosine")
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].ID != "doc1" {
		t.Errorf("cosine nearest: got %q, want %q", results[0].ID, "doc1")
	}
}

func TestSearch_MemoryInvertedIndex(t *testing.T) {
	t.Parallel()

	index := metaengine.NewMemorySearchIndex()
	ctx := context.Background()

	index.Insert(ctx, "test", metaengine.IndexedText{ID: "d1", Content: "The quick brown fox"})
	index.Insert(ctx, "test", metaengine.IndexedText{ID: "d2", Content: "The lazy dog"})
	index.Insert(ctx, "test", metaengine.IndexedText{ID: "d3", Content: "A quick brown dog jumps"})

	results, err := index.Query(ctx, "test", "quick brown", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results matching 'quick brown', got %d", len(results))
	}

	// d1 has both terms, d3 has both terms
	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}

	if !ids["d1"] {
		t.Errorf("expected d1 in results")
	}

	if !ids["d3"] {
		t.Errorf("expected d3 in results")
	}
}

func TestVersionedStorage_InterfaceExists(t *testing.T) {
	t.Parallel()

	// Verify ExecuteAsOf exists and returns error for non-versioned engines
	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = store.ExecuteAsOf(ctx, "find_task", "task-1", time.Now())
	if err == nil {
		t.Error("expected error from ExecuteAsOf on non-versioned engine")
	}
}
