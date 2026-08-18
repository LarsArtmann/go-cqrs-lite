package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ── Vector demo: document embeddings with k-NN semantic search ──
//
// One event type (DocEmbedded) is folded into metaengine.Embedding records;
// the query declares a k-NN search and the planner routes it to the engine's
// vector ADT (brute-force on Memory/SQLite, ANN indexes elsewhere).

type DocEmbedded struct {
	ID     string
	Values []float32
}

type SemanticSearchQuery struct {
	Vector []float32
	Metric string
	K      int
}

func runVectorDemo(ctx context.Context) error {
	query := metaengine.Query[SemanticSearchQuery, metaengine.VectorResult](
		"semantic_search",
		metaengine.OnRecord(
			DocEmbedded{},
			func(_ record.Record, evt DocEmbedded) metaengine.Embedding {
				return metaengine.Embedding{ID: evt.ID, Values: evt.Values}
			},
		),
	)

	// art-dupl:accept standalone demo setup; each demo file is intentionally self-contained
	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, query)
	if err != nil {
		return fmt.Errorf("plan vector: %w", err)
	}

	defer func() { _ = store.Close() }()

	docs := []DocEmbedded{
		{ID: "go-basics", Values: []float32{1, 0, 0}},
		{ID: "cqrs-guide", Values: []float32{0, 1, 0}},
		{ID: "go-cqrs", Values: []float32{1, 1, 0}},
	}

	for _, doc := range docs {
		if err := store.Apply(ctx, "DocEmbedded", doc); err != nil {
			return fmt.Errorf("apply embedding: %w", err)
		}
	}

	const nearest = 2

	results, err := metaengine.VectorExecuteTyped[SemanticSearchQuery](
		ctx, store,
		SemanticSearchQuery{Vector: []float32{1, 0, 0}, Metric: "euclidean", K: nearest},
	)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	fmt.Printf("%d nearest documents to {1,0,0} (euclidean):\n", nearest)

	for _, result := range results {
		fmt.Printf("  %-12s distance=%.3f\n", result.ID, result.Distance)
	}

	return nil
}
