package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- Test event + query types for Vector ADT end-to-end pipeline ---

type DocEmbedded struct {
	ID     string
	Values []float32
}

type SemanticSearchInput struct {
	Vector []float32
	Metric string
	K      int
}

type FullTextSearchInput struct {
	Query string
	Limit int
}

type DocIndexed struct {
	ID      string
	Content string
}

func TestVectorFoldPipeline_EndToEnd(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[SemanticSearchInput, metaengine.VectorResult](
			"semantic_search",
			metaengine.On(DocEmbedded{}, func(e DocEmbedded) metaengine.Embedding {
				return metaengine.Embedding{ID: e.ID, Values: e.Values}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Apply events to build the vector index
	if err := store.Apply(ctx, "DocEmbedded", DocEmbedded{ID: "a", Values: []float32{1, 0, 0}}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "DocEmbedded", DocEmbedded{ID: "b", Values: []float32{0, 1, 0}}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "DocEmbedded", DocEmbedded{ID: "c", Values: []float32{1, 1, 0}}); err != nil {
		t.Fatal(err)
	}

	// Query: find 2 nearest neighbors of {1, 0, 0} using euclidean distance
	results, err := metaengine.VectorExecuteTyped[SemanticSearchInput](
		ctx, store,
		SemanticSearchInput{Vector: []float32{1, 0, 0}, Metric: "euclidean", K: 2},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Nearest should be "a" (distance 0 from {1,0,0})
	if results[0].ID != "a" {
		t.Errorf("nearest: got %q, want %q", results[0].ID, "a")
	}
}

func TestSearchFoldPipeline_EndToEnd(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[FullTextSearchInput, metaengine.SearchResult](
			"full_text_search",
			metaengine.On(DocIndexed{}, func(e DocIndexed) metaengine.IndexedText {
				return metaengine.IndexedText{ID: e.ID, Content: e.Content}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Index some documents
	docs := []DocIndexed{
		{ID: "d1", Content: "The quick brown fox"},
		{ID: "d2", Content: "The lazy dog"},
		{ID: "d3", Content: "A quick brown dog jumps"},
	}

	for _, d := range docs {
		if err := store.Apply(ctx, "DocIndexed", d); err != nil {
			t.Fatal(err)
		}
	}

	// Search for "quick brown" — should match d1 and d3
	results, err := metaengine.SearchExecuteTyped[FullTextSearchInput](
		ctx, store,
		FullTextSearchInput{Query: "quick brown", Limit: 10},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}

	if !ids["d1"] || !ids["d3"] {
		t.Errorf("expected d1 and d3, got %v", ids)
	}
}

func TestVectorFoldPipeline_Classification(t *testing.T) {
	t.Parallel()

	// Verify the planner classifies an Embedding-returning fold as ADTVector
	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[SemanticSearchInput, metaengine.VectorResult](
			"vec_classify",
			metaengine.On(DocEmbedded{}, func(e DocEmbedded) metaengine.Embedding {
				return metaengine.Embedding{ID: e.ID, Values: e.Values}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].ADT != metaengine.ADTVector {
		t.Errorf("expected ADT %q, got %q", metaengine.ADTVector, collections[0].ADT)
	}

	if collections[0].ReadPattern != metaengine.ReadVectorSearch {
		t.Errorf("expected pattern %q, got %q",
			metaengine.ReadVectorSearch, collections[0].ReadPattern)
	}
}

func TestSearchFoldPipeline_Classification(t *testing.T) {
	t.Parallel()

	// Verify the planner classifies an IndexedText-returning fold as ADTSearch
	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[FullTextSearchInput, metaengine.SearchResult](
			"search_classify",
			metaengine.On(DocIndexed{}, func(e DocIndexed) metaengine.IndexedText {
				return metaengine.IndexedText{ID: e.ID, Content: e.Content}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].ADT != metaengine.ADTSearch {
		t.Errorf("expected ADT %q, got %q", metaengine.ADTSearch, collections[0].ADT)
	}

	if collections[0].ReadPattern != metaengine.ReadFullTextSearch {
		t.Errorf("expected pattern %q, got %q",
			metaengine.ReadFullTextSearch, collections[0].ReadPattern)
	}
}
