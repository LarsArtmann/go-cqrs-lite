package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C030: Infinite loop without context cancellation in handler code.
// Detects `for {}` or `for true {}` loops that lack a `case <-ctx.Done()`
// branch in any select within the loop body. Such loops cannot be cancelled
// and will leak goroutines on shutdown.
//
//nolint:ireturn // factory returns public interface
func NewC030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C030-no-ctx-cancel-in-loop",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					forStmt, ok := n.(*ast.ForStmt)
					if !ok {
						return true
					}

					// Only flag infinite loops (no condition or condition is `true`).
					if !isInfiniteLoop(forStmt) {
						return true
					}

					// Check if the loop body has a select with ctx.Done().
					if loopHasCtxDone(forStmt.Body) {
						return true
					}

					pos := ctx.Fset.Position(forStmt.Pos())

					f, err := finding.NewBuilder(
						"C030", toolName,
						"Infinite loop without context cancellation — "+
							"goroutine cannot be stopped on shutdown",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Add `case <-ctx.Done(): return` in a select inside the loop").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func isInfiniteLoop(stmt *ast.ForStmt) bool {
	if stmt.Cond == nil {
		return true
	}

	if ident, ok := stmt.Cond.(*ast.Ident); ok {
		return ident.Name == "true"
	}

	return false
}

func loopHasCtxDone(body *ast.BlockStmt) bool {
	found := false

	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}

		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "Done" {
			if recv, ok := sel.X.(*ast.Ident); ok && recv.Name == "ctx" {
				found = true
				return false
			}
		}

		return true
	})

	return found
}
