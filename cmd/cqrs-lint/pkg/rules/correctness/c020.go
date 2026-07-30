package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects panic() calls inside functions passed to bus.Subscribe or
// bus.SubscribeAll (function literals or method references). A panic in a
// bus handler can crash the entire bus or projection host.
//
//nolint:ireturn // factory returns public interface
func NewC020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C020-panic-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
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

					method := sel.Sel.Name
					if method != "Subscribe" && method != "SubscribeAll" {
						return true
					}

					// Find the handler function argument (last arg or second arg).
					handlerExpr := findHandlerArg(call)
					if handlerExpr == nil {
						return true
					}

					// Case 1: inline function literal.
					if lit, ok := handlerExpr.(*ast.FuncLit); ok {
						reportPanicsInFunc(ctx, lit.Body, &findings)
						return true
					}

					// Case 2: function reference (ident) — find the FuncDecl.
					if ident, ok := handlerExpr.(*ast.Ident); ok {
						fn := findFuncDecl(ctx, ident.Name, gf)
						if fn != nil && fn.Body != nil {
							reportPanicsInFunc(ctx, fn.Body, &findings)
						}
					}

					// Case 3: method value (x.Method) — find the method on the type.
					if msel, ok := handlerExpr.(*ast.SelectorExpr); ok {
						fn := findMethodDecl(ctx, msel)
						if fn != nil && fn.Body != nil {
							reportPanicsInFunc(ctx, fn.Body, &findings)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// findHandlerArg returns the function argument of a Subscribe/SubscribeAll call.
func findHandlerArg(call *ast.CallExpr) ast.Expr {
	if len(call.Args) == 0 {
		return nil
	}

	// Subscribe("type", handler) → last arg
	// SubscribeAll(handler) → last arg
	return call.Args[len(call.Args)-1]
}

// reportPanicsInFunc finds all panic() calls in a function body and appends findings.
func reportPanicsInFunc(ctx *analyzer.AnalysisContext, body *ast.BlockStmt, findings *[]finding.Finding) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "panic" {
			return true
		}

		pos := ctx.Fset.Position(call.Pos())

		f, err := finding.NewBuilder(
			"C020", toolName,
			"panic() in event handler — will crash the bus/projection host",
			finding.SeverityError,
			finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
		).
			WithCategory(finding.CategoryCorrectness).
			WithConfidence(finding.ConfidenceHigh).
			WithFixStrategy(finding.FixStrategySuggest).
			WithSuggestion("Return an error or log+skip instead of panicking").
			WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
			Build()
		if err != nil {
			return true
		}

		*findings = append(*findings, f)
		return true
	})
}

// findFuncDecl searches all GoFiles for a top-level func with the given name.
func findFuncDecl(ctx *analyzer.AnalysisContext, name string, currentFile *analyzer.GoFile) *ast.FuncDecl {
	for _, gf := range ctx.GoFiles {
		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}

			if fn.Name.Name == name {
				return fn
			}
		}
	}

	return nil
}

// findMethodDecl searches for a method declaration matching a selector expression.
func findMethodDecl(ctx *analyzer.AnalysisContext, sel *ast.SelectorExpr) *ast.FuncDecl {
	methodName := sel.Sel.Name
	recvType := analyzer.BaseTypeName(sel.X)

	for _, gf := range ctx.GoFiles {
		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Recv == nil {
				continue
			}

			if fn.Name.Name == methodName && analyzer.BaseTypeName(fn.Recv.List[0].Type) == recvType {
				return fn
			}
		}
	}

	return nil
}
