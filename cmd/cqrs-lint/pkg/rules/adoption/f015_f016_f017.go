package adoption

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
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
			if usesMetaengine(ctx) {
				return nil, nil
			}

			// Count query registrations as a proxy for query complexity.
			queryCount := 0
			queryCount += countCalls(ctx, "query", "RegisterTyped")
			queryCount += countCalls(ctx, "query", "Register")

			if queryCount < 3 {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F015",
				"Project has "+itoa(queryCount)+
					" query registrations but metaengine is not used — "+
					"cost-based planning, FilterOnField/SortOnField SQL pushdown, "+
					"and layout optimization are unavailable",
				"Import the metaengine module and use metaengine.Query[Q,R]() to declare "+
					"queries with FilterOnField/SortOnField pushdown. For SQLite: "+
					"metaengine.PlanFromSQLite(dsn, queries...) is a one-call setup. "+
					"The planner auto-selects the cheapest engine per query (Memory vs SQLite vs DuckDB).",
				pos, finding.ConfidenceLow,
			), nil
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
			if importsPath(ctx, "go-cqrs-lite/listing") {
				return nil, nil
			}

			if distinctAggregateCount(ctx) < 5 {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F016",
				"Project has "+itoa(distinctAggregateCount(ctx))+
					" aggregate types but listing module is not used — "+
					"stream status tracking is unavailable",
				"Import the listing module and use listing.StreamListing for "+
					"stream status tracking, tombstone detection, and stream "+
					"enumeration across aggregate types.",
				pos, finding.ConfidenceLow,
			), nil
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
			// Only relevant for projects with a distributed/async event bus —
			// in-memory buses don't have at-least-once delivery issues.
			if !ctx.FeatureProfile.HasAsyncBus {
				return nil, nil
			}

			if importsPath(ctx, "go-cqrs-lite/dedup") {
				return nil, nil
			}

			if !projectHasCallAny(ctx, "bus", "Subscribe", "SubscribeAll") {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "bus", "Subscribe")
			if !ok {
				pos, ok = firstCallPos(ctx, "bus", "SubscribeAll")
			}

			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(
				ctx,
				"F017",
				"bus.Subscribe/SubscribeAll is used but dedup module is not — "+
					"duplicate event delivery under at-least-once semantics is "+
					"not handled at stream boundaries",
				"Import the dedup module and use dedup.NewRing() for O(1) "+
					"fixed-capacity ID deduplication at stream boundaries. "+
					"Pair with idempotency middleware for full duplicate protection.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// countCalls counts calls to pkgName.funcName across non-test files.
func countCalls(ctx *analyzer.AnalysisContext, pkgName, funcName string) int {
	count := 0

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		astInspectCalls(gf.AST, func(call *ast.CallExpr) bool {
			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if sel.Sel.Name == funcName {
				count++
			}

			return true
		})
	}

	return count
}
