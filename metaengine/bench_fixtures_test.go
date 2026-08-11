package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Shared fixtures for coverage tests (bench_filter benchmarks moved to
// metaengine/bench). These types and helpers exercise the pushdown scan and
// typed reader paths that coverage_test.go relies on.

type benchItemResult struct {
	ID       string
	Status   string
	Priority int
}

type benchListInput struct {
	Status string
}

func benchFilterQuery() metaengine.QueryDecl[benchListInput, benchItemResult] {
	return metaengine.Query[benchListInput, benchItemResult](
		"bench_filter_scan",
		metaengine.OnRecord(
			benchItemResult{},
			func(_ record.Record, e benchItemResult) (string, benchItemResult) {
				return e.ID, e
			},
		),
		metaengine.FilterOnField[benchItemResult]("Status", metaengine.FilterEq),
		metaengine.SortOnField[benchItemResult]("Priority", true),
	)
}

func seedBenchStore(t testing.TB, store *metaengine.Store, n int) {
	t.Helper()
	ctx := context.Background()

	for i := range n {
		status := "open"
		if i%3 == 0 {
			status = "closed"
		}

		item := benchItemResult{
			ID:       fmt.Sprintf("item-%06d", i),
			Status:   status,
			Priority: i % 10,
		}

		if err := store.Apply(ctx, "benchItemResult", item); err != nil {
			t.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}
}

func setupBenchStore(
	tb testing.TB,
	n int,
	useSQLite bool,
) (*metaengine.Store, *metaengine.TypedReader[benchItemResult]) {
	tb.Helper()

	var engines []metaengine.Engine

	if useSQLite {
		eng, db := newSQLiteEngine()
		tb.Cleanup(func() { _ = db.Close() })
		engines = []metaengine.Engine{metaengine.NewMemoryEngine(), eng}
	} else {
		engines = []metaengine.Engine{metaengine.NewMemoryEngine()}
	}

	store, err := metaengine.Plan(engines, benchFilterQuery())
	if err != nil {
		tb.Fatalf("Plan: %v", err)
	}

	seedBenchStore(tb, store, n)

	reader := metaengine.NewReader[benchItemResult](store, "bench_filter_scan")

	return store, reader
}
