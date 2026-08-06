package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F022 detects CQRS projects with a SQL store that perform manual in-memory
// sorting (sort.Slice, slices.SortFunc) without using the metaengine module.
// Metaengine's SortOnField enables SQL ORDER BY pushdown with indexed access,
// avoiding loading all rows into Go memory for sorting.
//
// Only fires for SQL-backed stores (SQLite, Postgres, MySQL, DuckDB, Custom)
// because the pushdown requires a SQL engine. Memory and Pebble stores cannot
// push sort to the storage layer.
//
//nolint:ireturn // factory returns public interface
func NewF022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F022-manual-sort-no-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			if usesMetaengine(ctx) {
				return nil, nil
			}

			if !hasSQLStore(ctx) {
				return nil, nil
			}

			pos, ok := firstManualSortPos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F022",
				"Manual in-memory sorting (sort.Slice/slices.SortFunc) with a SQL "+
					"store but no metaengine — all rows loaded into Go memory for sorting",
				"Use metaengine.SortOnField for declarative ORDER BY pushdown. "+
					"Declare queries with metaengine.Query[Q,R](name, folds..., "+
					"metaengine.SortOnField[R](\"column\", true)) and the planner "+
					"pushes sort to the SQL engine with indexed access. "+
					"For SQLite: sqliteengine.PlanFromDSN(dsn, queries...) is a one-call setup.",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// hasSQLStore reports whether the project's detected store is SQL-backed
// (capable of ORDER BY / WHERE pushdown). Delegates to StoreKind.IsSQL
// so store-type classification lives in one place.
func hasSQLStore(ctx *analyzer.AnalysisContext) bool {
	return ctx.FeatureProfile.Store.IsSQL()
}

// firstManualSortPos returns the position of the first manual sort call
// (sort.Slice, sort.SliceStable, slices.SortFunc, slices.SortStableFunc,
// slices.Sort) in any non-test file.
func firstManualSortPos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			for _, p := range manualSortPatterns {
				if pkg.Name == p.pkg && sel.Sel.Name == p.name {
					hit = call
					return false
				}
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
