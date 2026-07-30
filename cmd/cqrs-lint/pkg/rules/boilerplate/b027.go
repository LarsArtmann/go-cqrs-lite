package boilerplate

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B027: Hardcoded stream-type string literals.
// Detects bare string literals (e.g. "User", "Order") passed as the
// streamType argument to event.New, repo.Execute, repo.Load, etc.
// These should be constants for consistency and refactor safety.
//
//nolint:ireturn // factory returns public interface
func NewB027Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B027-hardcoded-stream-type",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			// streamType is the 3rd arg (index 2) for event.New / NewEvent.
			// For repo.Execute/Load it's the 3rd arg (index 2).
			cqrsCallsWithStreamType := map[string]int{
				"New":            2,
				"NewEvent":       2,
				"NewEvents":      2,
				"Execute":        2,
				"Load":           2,
				"LoadAtVersion":  2,
				"LoadAtTime":     2,
			}

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

					argIdx, known := cqrsCallsWithStreamType[sel.Sel.Name]
					if !known {
						return true
					}

					if len(call.Args) <= argIdx {
						return true
					}

					lit, ok := call.Args[argIdx].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"B027", toolName,
						"Hardcoded stream-type string literal — "+
							"use a constant (e.g. const StreamType = id.StreamType(\"User\")) for consistency",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Extract to a package-level constant: const streamType = id.StreamType(\"User\")").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
