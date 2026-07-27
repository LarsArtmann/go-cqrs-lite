package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C015: Unchecked Close() — resource leak risk.
//
// Detects calls to .Close() whose error return is discarded (either as a
// bare statement or assigned to _). Ignoring Close errors can mask data
// corruption, incomplete flushes, or file descriptor leaks.
//
//nolint:ireturn // factory returns public interface
func NewC015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C015-unchecked-close",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					// Pattern 1: bare expression statement x.Close()
					exprStmt, ok := n.(*ast.ExprStmt)
					if ok {
						call, ok := exprStmt.X.(*ast.CallExpr)
						if ok && isCloseCall(call) {
							reportUncheckedClose(ctx, &findings, call)
						}

						return true
					}

					// Pattern 2: _ = x.Close()
					assignStmt, ok := n.(*ast.AssignStmt)
					if ok && len(assignStmt.Lhs) == 1 {
						ident, ok := assignStmt.Lhs[0].(*ast.Ident)
						if ok && ident.Name == "_" {
							if call, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok &&
								isCloseCall(call) {
								reportUncheckedClose(ctx, &findings, call)
							}
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isCloseCall returns true if the call expression is a selector ending in "Close".
func isCloseCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Close"
}

func reportUncheckedClose(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
) {
	pos := ctx.Fset.Position(call.Pos())

	f, err := finding.NewBuilder(
		"C015",
		toolName,
		fmt.Sprintf("unchecked Close() at %s — error is discarded, resource leak risk", pos.String()),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion("Handle the error: if err := x.Close(); err != nil { return ... }, or defer func() { _ = x.Close() }() if truly ignorable with a comment explaining why.").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}
