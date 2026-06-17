package indexing_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
)

func ExampleIndex_DDL() {
	idx := indexing.Index{
		Name:    "idx_events_cursor",
		Table:   "events",
		Columns: []string{"occurred_at", "id"},
		Reason:  "cursor pagination for ReadFrom",
	}

	fmt.Println(idx.DDL())

	// Output:
	// CREATE INDEX IF NOT EXISTS idx_events_cursor ON events(occurred_at, id);
}

func ExampleRecommendedCQRSIndexes() {
	idxs := indexing.RecommendedCQRSIndexes()
	fmt.Println(len(idxs) > 0)

	// Output:
	// true
}

func ExampleAdvisor_AnalyzeQuery() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchema(context.Background(), db)

	advisor := indexing.NewAdvisor(db)
	recs, _ := advisor.AnalyzeQuery(context.Background(),
		"SELECT * FROM events WHERE id = ?", "dummy-id")

	// Primary-key lookup should use an index, so no recommendations.
	fmt.Println(len(recs) == 0)

	// Output:
	// true
}

func ExampleAutoIndexer_ApplyCQRSIndexes() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchema(context.Background(), db)

	auto := indexing.NewAutoIndexer(db)
	auto.Enable()
	_ = auto.ApplyCQRSIndexes(context.Background())

	advisor := indexing.NewAdvisor(db)
	_ = advisor.ExistingIndexes(context.Background())

	fmt.Println(advisor.HasIndex("idx_events_cursor"))

	// Output:
	// true
}

func ExampleInitSchemaWithIndexesAndOptimizations() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = turso.InitSchemaWithIndexesAndOptimizations(context.Background(), db)

	fmt.Println("ready")

	// Output:
	// ready
}

func ExampleApplyOptimizations() {
	db, _ := turso.OpenTemp("")
	defer func() { _ = db.Close() }()

	_ = indexing.ApplyOptimizations(context.Background(), db)

	fmt.Println("optimized")

	// Output:
	// optimized
}
