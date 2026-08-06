package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F023 detects CQRS projects with a SQL store that perform manual in-memory
// filtering (for-range + if + append) without using the metaengine module.
// Metaengine's FilterOnField enables SQL WHERE pushdown with indexed access,
// avoiding loading all rows into Go memory for filtering.
//
// Only fires for SQL-backed stores (SQLite, Postgres, MySQL, DuckDB, Custom)
// because the pushdown requires a SQL engine.
//
//nolint:ireturn // factory returns public interface
func NewF023Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F023-manual-filter-no-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			if usesMetaengine(ctx) {
				return nil, nil
			}

			if !hasSQLStore(ctx) {
				return nil, nil
			}

			pos, ok := firstManualFilterPos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F023",
				"Manual in-memory filtering (for-range + if + append) with a SQL "+
					"store but no metaengine — all rows loaded into Go memory for filtering",
				"Use metaengine.FilterOnField for declarative WHERE-clause pushdown. "+
					"Declare queries with metaengine.Query[Q,R](name, folds..., "+
					"metaengine.FilterOnField[R](\"column\", metaengine.FilterEq, value)) "+
					"and the planner pushes the filter to the SQL engine. "+
					"For SQLite: metaengine.PlanFromSQLite(dsn, queries...) is a one-call setup.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F024 detects CQRS projects with a SQL store that perform manual pagination
// via slice expressions (results[offset:offset+limit]) without using the
// metaengine module. Metaengine provides cursor-based pagination via WithLimit
// and WithCursor, which pushes LIMIT/OFFSET to the SQL engine.
//
// Only fires for SQL-backed stores.
//
//nolint:ireturn // factory returns public interface
func NewF024Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F024-manual-pagination-no-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			if usesMetaengine(ctx) {
				return nil, nil
			}

			if !hasSQLStore(ctx) {
				return nil, nil
			}

			pos, ok := firstManualPaginationPos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F024",
				"Manual pagination (slice[offset:offset+limit]) with a SQL store "+
					"but no metaengine — all rows loaded into Go memory before slicing",
				"Use metaengine cursor pagination: WithLimit(n) pushes LIMIT to the "+
					"SQL engine, and WithCursor/WithCursorString enable encoded cursor "+
					"pagination without OFFSET scans. Declare queries with "+
					"metaengine.Query[Q,R](name, folds...) and read via "+
					"metaengine.NewReader[R](store, collection).",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// F025 detects CQRS projects with a SQL store that perform manual counting or
// aggregation (for-range + count++/sum +=) without using the metaengine Counter
// ADT. Metaengine's CounterIncrement enables pre-materialized O(1) counts via
// INSERT ON CONFLICT DO UPDATE, avoiding full-table scans for every count.
//
// Only fires for SQL-backed stores.
//
//nolint:ireturn // factory returns public interface
func NewF025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F025-manual-count-no-counter-adt",
		func(_ context.Context) ([]finding.Finding, error) {
			if usesMetaengine(ctx) {
				return nil, nil
			}

			if !hasSQLStore(ctx) {
				return nil, nil
			}

			pos, ok := firstManualAggregationPos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F025",
				"Manual count/aggregation (for-range + count++/sum +=) with a SQL "+
					"store but no metaengine — full collection scanned for every count",
				"Use the metaengine Counter ADT for pre-materialized O(1) counts. "+
					"Declare a counter query with metaengine.Query[Q,R](name, "+
					"metaengine.On(Event{}, func(e Event) metaengine.Delta{ "+
					"return metaengine.Delta{e.Status: +1} })) and read via "+
					"metaengine.ExecuteTyped[Q, map[string]int64](ctx, store, input).",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// firstManualFilterPos returns the position of the first "filter into a new
// slice" idiom: a for/range loop whose body contains an if-statement whose
// body contains an append() call.
func firstManualFilterPos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			body := loopBody(n)
			if body == nil {
				return true
			}

			if filterAppendInBlock(body) {
				hit = n
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// firstManualPaginationPos returns the position of the first slice expression
// whose indices reference pagination-like variables (offset, limit, start,
// end, page, size, skip, take).
func firstManualPaginationPos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit *ast.SliceExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			slice, ok := n.(*ast.SliceExpr)
			if !ok {
				return true
			}

			if sliceHasPaginationVar(slice.Low) || sliceHasPaginationVar(slice.High) {
				hit = slice
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// firstManualAggregationPos returns the position of the first "manual count or
// sum" idiom: a for/range loop whose body contains an increment (++) or
// compound assignment (+=, -=).
func firstManualAggregationPos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			body := loopBody(n)
			if body == nil {
				return true
			}

			if hasAggregationStmt(body) {
				hit = n
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
