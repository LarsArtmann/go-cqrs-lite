package adoption

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F020 detects projects using metaengine.SortOn (closure-based sort)
// where metaengine.SortOnField (declarative) enables SQL ORDER BY pushdown.
// Closure-based sorts force in-memory sorting of all rows; declarative sorts
// enable ORDER BY pushdown for indexed access instead of full scan + sort.
//
// Fires even when SortOnField is also used elsewhere (mixed usage) —
// the specific SortOn call is still suboptimal for that query. In the
// mixed case, confidence is lowered and the message acknowledges the pattern.
//
//nolint:ireturn // factory returns public interface
func NewF020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F020-metaengine-sorton-pushdown",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				pos, ok := firstCallPosIn(ctx.Fset, sc.files, "metaengine", "SortOn")
				if !ok {
					continue
				}

				suggestion := "Use metaengine.SortOnField for declarative sorts that enable " +
					"ORDER BY pushdown (indexed access instead of full scan + sort). " +
					"SortOnField accepts a column name and descending flag, " +
					"allowing the SQLite/Postgres engine to use column indexes."

				if hasCallIn(sc.files, "metaengine", "SortOnField") {
					out = append(out, singleInfoFinding(
						ctx,
						"F020",
						"mixed metaengine usage: SortOnField (pushdown) and SortOn (closure) "+
							"— the SortOn call still forces in-memory sorting for that query",
						suggestion, pos, finding.ConfidenceLow,
					)...)
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F020",
					"metaengine.SortOn uses closure-based sorting which prevents "+
						"SQL ORDER BY pushdown — queries scan all rows and sort in Go memory",
					suggestion, pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
		},
	)
}

// F021 detects metaengine query declarations with many folds on the same
// query, which creates write amplification: a single event triggers
// multiple engine writes. While some multi-fold queries are necessary,
// excessive folds per query indicate a design where batching or
// denormalization would reduce write pressure.
//
// Per-query analysis: each metaengine.Query call is inspected individually.
// Only queries with 3+ direct fold-constructor arguments trigger a finding.
// This avoids false positives when folds are spread across many queries
// (e.g., 5 queries with 1 fold each = no amplification, vs. 1 query with
// 5 folds = real amplification).
//
//nolint:ireturn // factory returns public interface
func NewF021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F021-metaengine-write-amplification",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !importsPathIn(sc.files, "go-cqrs-lite/metaengine") {
					continue
				}

				queries, totalFolds := findQueriesWithFoldsIn(ctx.Fset, sc.files)

				if len(queries) > 0 {
					out = append(out, perQueryFindings(ctx, queries)...)
					continue
				}

				if totalFolds < 3 {
					continue
				}

				pos, ok := firstCallPosIn(ctx.Fset, sc.files, "metaengine", "OnRecordTyped")
				if !ok {
					pos, ok = firstCallPosIn(ctx.Fset, sc.files, "metaengine", "OnTyped")
					if !ok {
						pos, ok = firstCallPosIn(ctx.Fset, sc.files, "metaengine", "OnRecord")
						if !ok {
							pos, ok = firstCallPosIn(ctx.Fset, sc.files, "metaengine", "On")
							if !ok {
								pos, ok = firstFilePosIn(ctx.Fset, sc.files)
								if !ok {
									continue
								}
							}
						}
					}
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F021",
					fmt.Sprintf(
						"metaengine has %d fold declarations — high write amplification "+
							"may degrade ingest throughput (each event triggers multiple engine writes)",
						totalFolds,
					),
					"Consider batching related folds into a single handler, "+
						"or using WithLatencyBudget to let the planner coalesce writes. "+
						"Each fold maps to a separate engine operation (MapSet, CounterIncrement, etc.); "+
						"3+ folds suggest the projection model may be over-decomposed.",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

// queryFoldCount holds per-query fold analysis results.
type queryFoldCount struct {
	pos       token.Position
	name      string
	foldCount int
}

// findQueriesWithFolds scans for metaengine.Query calls and counts the
// fold-constructor arguments directly nested in each call. Returns one
// entry per Query call plus the total count of fold constructors seen
// anywhere in non-test files.
func findQueriesWithFoldsIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (queries []queryFoldCount, totalFolds int) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "metaengine" {
				return true
			}

			isFold := isFoldConstructor(sel.Sel.Name)
			if isFold {
				totalFolds++
			}

			if sel.Sel.Name != "Query" {
				return true
			}

			name := "<unknown>"
			foldCount := 0

			for i, arg := range call.Args {
				if i == 0 {
					if lit, ok := arg.(*ast.BasicLit); ok {
						name = lit.Value
					}

					continue
				}

				argCall, ok := arg.(*ast.CallExpr)
				if !ok {
					continue
				}

				argSel, ok := analyzer.SelectorFromExpr(argCall.Fun)
				if !ok {
					continue
				}

				argPkg, ok := argSel.X.(*ast.Ident)
				if !ok || argPkg.Name != "metaengine" {
					continue
				}

				if isFoldConstructor(argSel.Sel.Name) {
					foldCount++
				}
			}

			queries = append(queries, queryFoldCount{
				pos:       fset.Position(call.Pos()),
				name:      name,
				foldCount: foldCount,
			})

			return true
		})
	}

	return queries, totalFolds
}

// perQueryFindings builds findings for queries with 3+ folds.
func perQueryFindings(ctx *analyzer.AnalysisContext, queries []queryFoldCount) []finding.Finding {
	var findings []finding.Finding

	for _, q := range queries {
		if q.foldCount < 3 {
			continue
		}

		findings = append(findings, singleInfoFinding(
			ctx,
			"F021",
			fmt.Sprintf(
				"metaengine query %s has %d fold declarations — high write amplification "+
					"may degrade ingest throughput (each event triggers multiple engine writes)",
				q.name, q.foldCount,
			),
			"Consider batching related folds into a single handler, "+
				"or using WithLatencyBudget to let the planner coalesce writes. "+
				"Each fold maps to a separate engine operation (MapSet, CounterIncrement, etc.); "+
				"3+ folds on a single query suggest the projection model may be over-decomposed.",
			q.pos, finding.ConfidenceLow,
		)...)
	}

	return findings
}
