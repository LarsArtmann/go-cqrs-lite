package boilerplate

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B028: Manual goroutine dispatch instead of deriver package.
// Detects `go func() { disp.Dispatch(ctx, cmd) }()` patterns inside
// projection/event handler code. The deriver package provides Idempotent
// dispatch with proper error propagation, deterministic command IDs, and
// event-type filtering.
//
//nolint:ireturn // factory returns public interface
func NewB028Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B028-manual-goroutine-dispatch",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					goStmt, ok := n.(*ast.GoStmt)
					if !ok {
						return true
					}

					// Check if the goroutine body calls Dispatch or Execute.
					var dispatchPos ast.Node

					ast.Inspect(goStmt.Call, func(inner ast.Node) bool {
						if dispatchPos != nil {
							return false
						}

						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							return true
						}

						if sel.Sel.Name == "Dispatch" || sel.Sel.Name == "Execute" {
							dispatchPos = call
							return false
						}

						return true
					})

					if dispatchPos == nil {
						return true
					}

					pos := ctx.Fset.Position(goStmt.Pos())

					f, err := finding.NewBuilder(
						"B028", toolName,
						"Manual goroutine dispatch of command — "+
							"deriver.AsHandler provides idempotent dispatch, error propagation, and event-type filtering",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use deriver.AsHandler(dispatcher) for synchronous, idempotent event-to-command dispatch").
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
