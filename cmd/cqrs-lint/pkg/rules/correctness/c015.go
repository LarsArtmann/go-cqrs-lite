package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"slices"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C015: Unchecked Close() — resource leak risk.
//
// Detects calls to .Close() whose error return is discarded (either as a
// bare statement or assigned to _). Ignoring Close errors can mask data
// corruption, incomplete flushes, or file descriptor leaks.
//
// NewC015Detector detects unchecked Close() calls that risk resource leaks.
// Suppressions:
//   - defer bodies (defer x.Close(), defer func(){ _ = x.Close() }())
//   - error-cleanup blocks: if-statements containing both the Close() and a
//     return statement (e.g., if err != nil { _ = db.Close(); return nil, err })
//   - cleanup callbacks: anonymous functions (FuncLit) where the Close error
//     cannot be propagated (e.g., t.Cleanup(func() { _ = b.Close() }))
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

				var ancestors []ast.Node

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					if n == nil {
						if len(ancestors) > 0 {
							ancestors = ancestors[:len(ancestors)-1]
						}

						return true
					}

					ancestors = append(ancestors, n)

					if isSuppressedClose(ancestors) {
						return true
					}

					if exprStmt, ok := n.(*ast.ExprStmt); ok {
						if call, ok := exprStmt.X.(*ast.CallExpr); ok && isCloseCall(call) {
							reportUncheckedClose(ctx, &findings, call)
						}

						return true
					}

					if assignStmt, ok := n.(*ast.AssignStmt); ok && len(assignStmt.Lhs) == 1 {
						if ident, ok := assignStmt.Lhs[0].(*ast.Ident); ok && ident.Name == "_" {
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

// isSuppressedClose returns true if the current position is inside a context
// where discarding Close() errors is the accepted cleanup pattern:
//   - inside a defer body
//   - inside an error-cleanup if-block (contains a return statement)
//   - inside an anonymous function (cleanup callback)
func isSuppressedClose(ancestors []ast.Node) bool {
	return isInDefer(ancestors) ||
		isInErrorCleanup(ancestors) ||
		isInCleanupCallback(ancestors)
}

// isInDefer returns true if any ancestor node is a DeferStmt, indicating the
// current node executes inside a deferred context where Close() discarding is
// the recommended cleanup pattern.
func isInDefer(ancestors []ast.Node) bool {
	for _, a := range ancestors {
		if _, ok := a.(*ast.DeferStmt); ok {
			return true
		}
	}

	return false
}

// isInErrorCleanup returns true if the nearest enclosing block is the body of
// an if-statement that contains a return statement. This pattern is the
// standard error-cleanup idiom:
//
//	if err != nil {
//	    _ = db.Close()  // best-effort cleanup before returning the real error
//	    return nil, err
//	}
//
// The Close error is intentionally discarded because the function is already
// returning a more important error from a prior operation.
func isInErrorCleanup(ancestors []ast.Node) bool {
	var block *ast.BlockStmt
	var blockParent ast.Node

	for i := range slices.Backward(ancestors) {
		if b, ok := ancestors[i].(*ast.BlockStmt); ok {
			block = b
			if i > 0 {
				blockParent = ancestors[i-1]
			}

			break
		}
	}

	if block == nil {
		return false
	}

	if _, ok := blockParent.(*ast.IfStmt); !ok {
		return false
	}

	for _, stmt := range block.List {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			return true
		}
	}

	return false
}

// isInCleanupCallback returns true if any ancestor is a FuncLit (anonymous
// function). This covers cleanup callbacks where the Close error cannot be
// propagated because the function signature does not return an error:
//
//	t.Cleanup(func() { _ = b.Close() })
//	closeFn := func() { _ = backend.Close(); _ = sqlDB.Close() }
func isInCleanupCallback(ancestors []ast.Node) bool {
	for _, a := range ancestors {
		if _, ok := a.(*ast.FuncLit); ok {
			return true
		}
	}

	return false
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
