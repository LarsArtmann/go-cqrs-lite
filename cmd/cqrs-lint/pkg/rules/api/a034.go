package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// A034: Using the untyped metaengine.Execute instead of ExecuteTyped.
//
// metaengine.Execute returns an `any` result, requiring a runtime type
// assertion that can panic. metaengine.ExecuteTyped[Q, R] returns a typed
// result with compile-time safety, eliminating the assertion and the panic
// risk.
//
//nolint:ireturn // factory returns public interface
func NewA034Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A034-metaengine-execute-untyped",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasMetaengine {
				return nil, nil
			}

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

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if sel.Sel.Name != "Execute" {
						return true
					}

					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						return true
					}

					if !lintutil.QualifierResolvesTo(
						gf.AST,
						ident.Name,
						"go-cqrs-lite/metaengine",
					) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A034", toolName,
						"metaengine.Execute returns an untyped result (any) — "+
							"requires a runtime type assertion that can panic",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use metaengine.ExecuteTyped[Q, R](ctx, store, input) " +
							"for a compile-time typed result without assertions").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)

					return true
				})
			}

			return findings, nil
		},
	)
}
