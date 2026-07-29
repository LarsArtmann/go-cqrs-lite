package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C016: context.Background() or context.TODO() in handlers.
//
// Detects usage of context.Background() or context.TODO() inside functions
// that receive a context.Context parameter. Handler functions should propagate
// the caller's context for cancellation, timeouts, and tracing — not create a
// fresh detached context that ignores the caller's lifecycle.
//
//nolint:ireturn // factory returns public interface
func NewC016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C016-background-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fnDecl, ok := n.(*ast.FuncDecl)
					if !ok {
						return true
					}

					if !hasContextParam(fnDecl) {
						return true
					}

					// Walk the function body for context.Background()/TODO() calls.
					if fnDecl.Body == nil {
						return true
					}

					ast.Inspect(fnDecl.Body, func(inner ast.Node) bool {
						// Don't flag nested function literals — they may have
						// their own (or no) context parameter.
						if _, ok := inner.(*ast.FuncLit); ok {
							return false
						}

						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						ident, ok := sel.X.(*ast.Ident)
						if !ok || ident.Name != "context" {
							return true
						}

						if sel.Sel.Name != "Background" && sel.Sel.Name != "TODO" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C016",
							toolName,
							fmt.Sprintf(
								"context.%s() in handler %s at %s — discards caller context (cancellation, timeouts, tracing lost)",
								sel.Sel.Name, fnDecl.Name.Name, pos.String(),
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Use the context.Context parameter passed to the handler. If you need a detached context for a background task, extract it explicitly and document why the caller's context cannot be used.").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

// hasContextParam returns true if the function declaration has a parameter
// of type context.Context (by name convention "ctx" or by type).
func hasContextParam(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}

	for _, field := range fn.Type.Params.List {
		if isContextType(field.Type) {
			return true
		}
	}

	return false
}

func isContextType(expr ast.Expr) bool {
	// Direct: context.Context
	sel, ok := expr.(*ast.SelectorExpr)
	if ok {
		ident, ok := sel.X.(*ast.Ident)

		return ok && ident.Name == "context" && sel.Sel.Name == "Context"
	}

	// Pointer or ellipsis: recurse one level
	if star, ok := expr.(*ast.StarExpr); ok {
		return isContextType(star.X)
	}

	if ell, ok := expr.(*ast.Ellipsis); ok {
		return isContextType(ell.Elt)
	}

	return false
}
