package performance

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P008: Projection host WithBatchSize not set.
// Detects projectionhost.New calls without WithBatchSize. The batch size
// controls how many events are fetched per round-trip from the journal.
// For large event streams, the default batch size can bottleneck throughput.
//
//nolint:ireturn // factory returns public interface
func NewP008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P008-projectionhost-no-batchsize",
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
					if !ok || sel.Sel.Name != "New" {
						return true
					}

					if analyzer.SelectorPackage(sel) != "projectionhost" {
						return true
					}

					if callHasOption(call, "WithBatchSize") {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"P008", toolName,
						"projectionhost.New without WithBatchSize — "+
							"default batch size may bottleneck throughput on large event streams",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Add projectionhost.WithBatchSize(n) — "+
							"tune based on event payload size and journal latency").
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
