package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F020 detects projects using metaengine.SortOn (closure-based sort)
// where metaengine.SortOnField (declarative) enables SQL ORDER BY pushdown.
// Closure-based sorts force in-memory sorting of all rows; declarative sorts
// enable ORDER BY pushdown for indexed access instead of full scan + sort.
//
//nolint:ireturn // factory returns public interface
func NewF020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F020-metaengine-sorton-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			if !usesMetaengine(ctx) {
				return nil, nil
			}

			if projectHasCall(ctx, "metaengine", "SortOnField") {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "metaengine", "SortOn")
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F020",
				"metaengine.SortOn uses closure-based sorting which prevents "+
					"SQL ORDER BY pushdown — queries scan all rows and sort in Go memory",
				"Use metaengine.SortOnField for declarative sorts that enable "+
					"ORDER BY pushdown (indexed access instead of full scan + sort). "+
					"SortOnField accepts a column name and descending flag, "+
					"allowing the SQLite/Postgres engine to use column indexes.",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// F021 detects metaengine query declarations with many folds on the same
// event type, which creates write amplification: a single event triggers
// multiple engine writes across collections. While some multi-collection
// updates are necessary, excessive folds per event type indicate a design
// where batching or denormalization would reduce write pressure.
//
//nolint:ireturn // factory returns public interface
func NewF021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F021-metaengine-write-amplification",
		func(_ context.Context) ([]finding.Finding, error) {
			if !usesMetaengine(ctx) {
				return nil, nil
			}

			// Detect multiple OnTyped/On calls — each fold is a write.
			// If there are 5+ fold declarations without a Batch wrapper,
			// warn about write amplification.
			foldCount := countCalls(ctx, "metaengine", "OnTyped") +
				countCalls(ctx, "metaengine", "On")

			if foldCount < 5 {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "metaengine", "OnTyped")
			if !ok {
				pos, ok = firstCallPos(ctx, "metaengine", "On")
				if !ok {
					pos, ok = firstFilePos(ctx)
					if !ok {
						return nil, nil
					}
				}
			}

			return singleInfoFinding(
				ctx,
				"F021",
				"metaengine query has many fold declarations — high write amplification "+
					"may degrade ingest throughput (each event triggers multiple engine writes)",
				"Consider batching related folds into a single handler, "+
					"or using WithLatencyBudget to let the planner coalesce writes. "+
					"Each fold maps to a separate engine operation (MapSet, CounterIncrement, etc.); "+
					"5+ folds per query suggest the projection model may be over-decomposed.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
