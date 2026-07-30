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
					handlerExpr := analyzer.FindHandlerArg(call)
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
						fn := analyzer.FindFuncDecl(ctx, ident.Name)
						if fn != nil && fn.Body != nil {
							reportPanicsInFunc(ctx, fn.Body, &findings)
						}
					}

					// Case 3: method value (x.Method) — find the method on the type.
					if msel, ok := handlerExpr.(*ast.SelectorExpr); ok {
						fn := analyzer.FindMethodDecl(ctx, msel)
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

