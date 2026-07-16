package api

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A007: Dual model (OO aggregate + functional decider).
// Detects projects that use both OO-style aggregates and functional deciders.
func NewA007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A007-dual-model-oo-functional",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasOO := false
			hasFunctional := false

			var firstOO analyzer.DeciderInfo

			for _, d := range ctx.Registry.Deciders {
				if d.IsOO {
					hasOO = true

					if firstOO.File == "" {
						firstOO = d
					}
				}
			}
			// Check for functional decider usage (decider.Decider[State]{...}).
			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					lit, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}

					typeStr := analyzer.ExprString(lit.Type)
					if strings.Contains(typeStr, "decider.Decider") ||
						strings.Contains(typeStr, "Decider[") {
						hasFunctional = true
					}

					return true
				})
			}

			if hasOO && hasFunctional {
				f, err := finding.NewBuilder(
					"A007", toolName,
					"Project uses both OO-style aggregates and functional deciders — pick one model for consistency",
					finding.SeverityError,
					finding.Pos(finding.FilePath(firstOO.File), firstOO.Pos.Line, firstOO.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Use the functional decider.Decider[State] pattern (Initial + Apply) consistently — it's the recommended approach").
					WithSnippet(ctx.SourceLine(firstOO.File, firstOO.Pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}
