package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// C034: Goroutine without context cancellation.
// Detects `go func()` inside handler code without ctx propagation.
// A goroutine that doesn't receive ctx can outlive the parent handler,
// leading to resource leaks on shutdown.
//
//nolint:ireturn // factory returns public interface
func NewC034Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C034-goroutine-without-ctx",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					if !hasCtxParam(fn.Type) {
						return true
					}

					derivedCtxVars := findDerivedContextVars(fn.Body)

					ast.Inspect(fn.Body, func(inner ast.Node) bool {
						goStmt, ok := inner.(*ast.GoStmt)
						if !ok {
							return true
						}

						if goroutineHasCtxArg(goStmt, derivedCtxVars) {
							return true
						}

						if enclosingFuncHasShutdown(fn.Body) {
							return true
						}

						pos := ctx.Fset.Position(goStmt.Pos())

						f, err := finding.NewBuilder(
							"C034",
							toolName,
							"go func() without ctx — goroutine outlives parent handler, risk of resource leak on shutdown",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Pass ctx to the goroutine: go process(ctx, ...)").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

// goroutineHasCtxArg reports whether the go statement's function call passes
// a ctx-like argument. The derivedCtxVars set contains variable names assigned
// from context.WithCancel/WithTimeout/WithDeadline/WithValue calls — those are
// also valid context variables that properly propagate cancellation.
func goroutineHasCtxArg(goStmt *ast.GoStmt, derivedCtxVars map[string]bool) bool {
	if goStmt.Call == nil {
		return false
	}

	for _, arg := range goStmt.Call.Args {
		if id, ok := arg.(*ast.Ident); ok {
			name := strings.ToLower(id.Name)
			if name == "ctx" || name == "context" || derivedCtxVars[id.Name] {
				return true
			}
		}
	}

	// Check function literals that capture ctx or a derived context.
	if fnLit, ok := goStmt.Call.Fun.(*ast.FuncLit); ok && fnLit.Body != nil {
		hasCtxRef := false
		ast.Inspect(fnLit.Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				name := strings.ToLower(id.Name)
				if name == "ctx" || name == "context" || derivedCtxVars[id.Name] {
					hasCtxRef = true
					return false
				}
			}
			return true
		})
		return hasCtxRef
	}

	return false
}

// findDerivedContextVars scans the function body for variables assigned from
// context.WithCancel/WithTimeout/WithDeadline/WithValue calls. These variables
// are valid context values — passing them to a goroutine counts as ctx
// propagation. Returns a set of variable names.
func findDerivedContextVars(body *ast.BlockStmt) map[string]bool {
	if body == nil {
		return nil
	}

	derived := make(map[string]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "context" {
				continue
			}
			if !strings.HasPrefix(sel.Sel.Name, "With") {
				continue
			}

			// The first LHS variable is the derived context.
			if len(assign.Lhs) > 0 {
				if id, ok := assign.Lhs[0].(*ast.Ident); ok {
					derived[id.Name] = true
				}
			}
		}
		return true
	})

	return derived
}

// enclosingFuncHasShutdown reports whether the function body contains
// ctx.Done() or .Shutdown() calls, indicating standard graceful shutdown
// patterns where goroutines without ctx are intentional (e.g. HTTP server
// lifecycle: go server.ListenAndServe() + ctx.Done() + server.Shutdown()).
func enclosingFuncHasShutdown(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}

	hasShutdown := false

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "Done" || sel.Sel.Name == "Shutdown" {
			hasShutdown = true
			return false
		}

		return true
	})

	return hasShutdown
}
