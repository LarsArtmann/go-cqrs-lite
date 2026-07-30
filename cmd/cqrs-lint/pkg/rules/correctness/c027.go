package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects bus.Subscribe/SubscribeAll calls in a codebase that also uses
// projectionhost. If both are active, events may be processed twice — once
// by the projection host and once by the direct bus subscription.
//
// C027: Bus subscription started alongside projectionhost.
//
//nolint:ireturn // factory returns public interface
func NewC027Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C027-bus-subscription-alongside-projectionhost",
		func(_ context.Context) ([]finding.Finding, error) {
			if !codebaseUsesProjectionHost(ctx) {
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

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					method := sel.Sel.Name
					if method != "Subscribe" && method != "SubscribeAll" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C027", toolName,
						"bus."+method+"() alongside projectionhost — "+
							"events may be processed twice (once by host, once by subscription)",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Register projections with projectionhost only, " +
							"or use bus subscriptions only — not both for the same event types").
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

// codebaseUsesProjectionHost returns true if any non-test file calls
// projectionhost.New.
func codebaseUsesProjectionHost(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if sel.Sel.Name == "New" && analyzer.SelectorPackage(sel) == "projectionhost" {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}
