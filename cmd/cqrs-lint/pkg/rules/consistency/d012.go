package consistency

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// D012: Raw print in handler/projection code.
// Detects fmt.Print*/Printf/Println and log.Print*/Fatal* inside functions
// that handle CQRS operations (bus.Subscribe/SubscribeAll handlers, projection
// Handle methods, decider decide functions). These should use structured
// logging (slog) for observability and log-level control.
//
//nolint:ireturn // factory returns public interface
func NewD012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D012-raw-print-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			rawPrintFuncs := map[string]bool{
				"Print": true, "Printf": true, "Println": true,
				"Fatal": true, "Fatalf": true, "Fatalln": true,
				"Panic": true, "Panicf": true, "Panicln": true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					if !isCQRSHandler(fn) {
						return true
					}

					ast.Inspect(fn.Body, func(inner ast.Node) bool {
						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if !rawPrintFuncs[sel.Sel.Name] {
							return true
						}

						ident, ok := sel.X.(*ast.Ident)
						if !ok || (ident.Name != "fmt" && ident.Name != "log") {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"D012", toolName,
							"Raw fmt/log print in CQRS handler — "+
								"use structured logging (slog) for observability and log-level control",
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryNaming).
							WithConfidence(finding.ConfidenceLow).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Replace with slog: logger.Info(\"message\", \"key\", value)").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

// isCQRSHandler returns true if the function looks like a CQRS handler
// (has a context.Context parameter and an event.Event or command parameter).
func isCQRSHandler(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}

	for _, field := range fn.Type.Params.List {
		// Check for context.Context parameter.
		if isContextType(field.Type) {
			return true
		}
		// Check for event.Event parameter.
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Event" || sel.Sel.Name == "Command" {
				return true
			}
		}
	}

	return false
}

func isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)

	return ok && ident.Name == "context" && sel.Sel.Name == "Context"
}
