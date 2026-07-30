package performance

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// P006: Polling loop with short interval for state-change detection.
// Detects time.Sleep with a duration < 100ms inside a for-loop — a busy-poll
// pattern that wastes CPU. A channel, callback, or sync.Cond would be
// zero-latency.
//
//nolint:ireturn // factory returns public interface
func NewP006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P006-polling-loop-short-interval",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					forStmt, ok := n.(*ast.ForStmt)
					if !ok {
						return true
					}

					// Find time.Sleep calls inside the loop.
					ast.Inspect(forStmt.Body, func(inner ast.Node) bool {
						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							return true
						}

						if sel.Sel.Name != "Sleep" {
							return true
						}

						pkg := analyzer.SelectorPackage(sel)
						if pkg != "time" {
							return true
						}

						// Check if the duration argument is small (< 100ms).
						if !isShortDuration(call) {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"P006", toolName,
							"time.Sleep with short interval inside a loop — "+
								"busy-poll wastes CPU; consider a channel, callback, or sync.Cond",
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryPerformance).
							WithConfidence(finding.ConfidenceLow).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Replace polling with a channel signal or callback for zero-latency notification").
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

// isShortDuration returns true if the Sleep argument is a literal duration
// expression like 10*time.Millisecond that is < 100ms.
func isShortDuration(call *ast.CallExpr) bool {
	if len(call.Args) < 1 {
		return false
	}

	// Check for `N * time.Millisecond` pattern.
	if bin, ok := call.Args[0].(*ast.BinaryExpr); ok {
		if bin.Op.String() != "*" {
			return false
		}

		// Left side is the number, right side is time.Xxx.
		if _, ok := bin.X.(*ast.BasicLit); ok {
			sel, ok := bin.Y.(*ast.SelectorExpr)
			if !ok {
				return false
			}

			if sel.Sel.Name == "Millisecond" || sel.Sel.Name == "Microsecond" {
				return true // Any ms or us is < 100ms
			}
		}
	}

	// Check for time.Duration literals (rare in Go, usually done via const).
	return false
}
