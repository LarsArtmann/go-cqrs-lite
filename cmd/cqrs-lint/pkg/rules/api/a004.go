package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// A004: Untyped dispatch register.
// Detects dispatcher.Register with type assertion inside the handler.
//
//nolint:ireturn // factory returns public interface
func NewA004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A004-untyped-dispatch-register",
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

					if sel.Sel.Name != "Register" && sel.Sel.Name != "Handle" {
						return true
					}
					// Skip third-party Register/Handle APIs (http, grpc, mux, chi, …)
					// whose method name collides with CQRS but serves a different
					// purpose. Without this, any router/handler framework triggers the
					// rule whenever a closure argument uses a type assertion.
					if lintutil.IsNonCQRSRegisterPackage(analyzer.SelectorPackage(sel)) {
						return true
					}
					// Check if the handler function literal contains a type assertion.
					for _, arg := range call.Args {
						funcLit, ok := arg.(*ast.FuncLit)
						if !ok {
							continue
						}

						hasTypeAssert := false

						ast.Inspect(funcLit.Body, func(nn ast.Node) bool {
							_, ok := nn.(*ast.TypeAssertExpr)
							if ok {
								hasTypeAssert = true

								return false
							}

							return true
						})

						if hasTypeAssert {
							pos := ctx.Fset.Position(call.Pos())

							f, err := finding.NewBuilder(
								"A004", toolName,
								"Untyped handler registration with type assertion — use RegisterTyped for compile-time type safety",
								finding.SeverityWarning,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceMedium).
								WithSuggestion("Use command.RegisterTyped or query.RegisterTyped to register typed handlers without runtime type assertions").
								WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
								Build()
							if err != nil {
								return true
							}

							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
