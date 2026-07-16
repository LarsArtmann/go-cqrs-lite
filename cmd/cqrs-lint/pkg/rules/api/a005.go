package api

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A005: Custom projection runner.
// Detects bus.SubscribeAll + manual switch without projectionhost.
func NewA005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A005-custom-projection-runner",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				hasSubscribeAll := false

				var subscribePos token.Position

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if sel.Sel.Name == "SubscribeAll" {
						hasSubscribeAll = true
						subscribePos = ctx.Fset.Position(call.Pos())
					}

					return true
				})

				if hasSubscribeAll {
					// Check if projectionhost is imported.
					usesProjectionHost := false

					for _, imp := range gf.AST.Imports {
						if imp.Path != nil && strings.Contains(imp.Path.Value, "projectionhost") {
							usesProjectionHost = true

							break
						}
					}

					if !usesProjectionHost {
						f, err := finding.NewBuilder(
							"A005", toolName,
							"Manual projection via bus.SubscribeAll — use projectionhost.Host for checkpoint persistence, dead-letter queues, and crash recovery",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(subscribePos.Filename), subscribePos.Line, subscribePos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Register projections with projectionhost.New(journal, checkpointStore) instead of manual bus.SubscribeAll + switch").
							WithSnippet(ctx.SourceLine(subscribePos.Filename, subscribePos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}
				}
			}

			return findings, nil
		},
	)
}
