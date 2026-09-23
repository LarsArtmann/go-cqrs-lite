package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				if !sc.profile.Store.IsSQL() {
					continue
				}

				pos, ok := firstManualFilterPosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F023",
					"Manual in-memory filtering (for-range + if + append) with a SQL "+
						"store but no metaengine — all rows loaded into Go memory for filtering",
					"Use metaengine.FilterOnField for declarative WHERE-clause pushdown. "+
						"Declare queries with metaengine.Query[Q,R](name, folds..., "+
						"metaengine.FilterOnField[R](\"column\", metaengine.FilterEq, value)) "+
						"and the planner pushes the filter to the SQL engine. "+
						"For SQLite: sqliteengine.PlanFromDSN(dsn, queries...) is a one-call setup.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				if !sc.profile.Store.IsSQL() {
					continue
				}

				pos, ok := firstManualPaginationPosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
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
				)...)
			}

			return out, nil
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
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				if !sc.profile.Store.IsSQL() {
					continue
				}

				pos, ok := firstManualAggregationPosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F025",
					"Manual count/aggregation (for-range + count++/sum +=) with a SQL "+
						"store but no metaengine — full collection scanned for every count",
					"Use the metaengine Counter ADT for pre-materialized O(1) counts. "+
						"Declare a counter query with metaengine.Query[Q,R](name, "+
						"metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Delta{ "+
						"return metaengine.Delta{e.Status: +1} })) and read via "+
						"metaengine.ExecuteTyped[Q, map[string]int64](ctx, store, input).",
					pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
		},
	)
}

func firstManualFilterPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	for _, gf := range files {
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
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

func firstManualPaginationPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	for _, gf := range files {
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
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

func firstManualAggregationPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	for _, gf := range files {
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
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
