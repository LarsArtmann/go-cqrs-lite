package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F015 detects projects with multiple query types that do not use the
// metaengine module. The metaengine provides a cost-based storage planner
// with universal ADT support, SQL pushdown (FilterOnField/SortOnField),
// and layout planning. It works with all backends — including SQLite via
// PlanFromSQLite / NewSQLiteEngineFromDSN.
//
//nolint:ireturn // factory returns public interface
func NewF015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F015-no-metaengine",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				queryCount := 0
				queryCount += countCallsIn(sc.files, "query", "RegisterTyped")
				queryCount += countCallsIn(sc.files, "query", "Register")

				if queryCount < 3 {
					continue
				}

				pos, ok := firstFilePosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F015",
					"Project has "+itoa(queryCount)+
						" query registrations but metaengine is not used — "+
						"cost-based planning, FilterOnField/SortOnField SQL pushdown, "+
						"and layout optimization are unavailable",
					"Import the metaengine module and use metaengine.Query[Q,R]() to declare "+
						"queries with FilterOnField/SortOnField pushdown. For SQLite: "+
						"sqliteengine.PlanFromDSN(dsn, queries...) is a one-call setup. "+
						"The planner auto-selects the cheapest engine per query (Memory vs SQLite vs DuckDB).",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

// F016 detects projects with many aggregate types that do not use the listing
// module. listing.StreamListing provides stream status tracking, tombstone
// detection, and stream enumeration.
//
//nolint:ireturn // factory returns public interface
func NewF016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F016-no-listing-module",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/listing") {
					continue
				}

				aggCount := distinctAggregateCountIn(ctx, sc.files)
				if aggCount < 5 {
					continue
				}

				pos, ok := firstFilePosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F016",
					"Project has "+itoa(aggCount)+
						" aggregate types but listing module is not used — "+
						"stream status tracking is unavailable",
					"Import the listing module and use listing.StreamListing for "+
						"stream status tracking, tombstone detection, and stream "+
						"enumeration across aggregate types.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

// F017 detects projects with bus subscriptions that do not use the dedup
// module. The dedup ring buffer provides O(1) fixed-capacity ID deduplication
// for stream boundaries under at-least-once delivery.
//
//nolint:ireturn // factory returns public interface
func NewF017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F017-no-dedup-module",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasAsyncBus {
					continue
				}

				if importsPathIn(sc.files, "go-cqrs-lite/dedup") {
					continue
				}

				if !hasCallIn(sc.files, "bus", "Subscribe", "SubscribeAll") {
					continue
				}

				pos, ok := firstCallPosIn(ctx.Fset, sc.files, "bus", "Subscribe")
				if !ok {
					pos, ok = firstCallPosIn(ctx.Fset, sc.files, "bus", "SubscribeAll")
				}

				if !ok {
					pos, ok = firstFilePosIn(ctx.Fset, sc.files)
					if !ok {
						continue
					}
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F017",
					"bus.Subscribe/SubscribeAll is used but dedup module is not — "+
						"duplicate event delivery under at-least-once semantics is "+
						"not handled at stream boundaries",
					"Import the dedup module and use dedup.NewRing() for O(1) "+
						"fixed-capacity ID deduplication at stream boundaries. "+
						"Pair with idempotency middleware for full duplicate protection.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}
