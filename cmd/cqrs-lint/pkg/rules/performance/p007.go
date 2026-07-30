package performance

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects manual retry loops that should use retry.Do. Flags hand-rolled
// retry logic, especially the bitshift backoff anti-pattern
// (baseBackoff << time.Duration(attempt)) which corrupts the Duration.
//
//nolint:ireturn // factory returns public interface
func NewP007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P007-manual-retry-loop",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					// Detect for/range loops with retry-like patterns.
					forStmt, ok := n.(*ast.ForStmt)
					if !ok {
						return true
					}

					// Check if the body contains a bitshift on a Duration.
					hasBitshiftDuration := false

					ast.Inspect(forStmt.Body, func(inner ast.Node) bool {
						binOp, ok := inner.(*ast.BinaryExpr)
						if !ok {
							return true
						}

						// << or >> on something involving time.Duration
						if binOp.Op == token.SHL || binOp.Op == token.SHR {
							if mentionsDuration(binOp) {
								hasBitshiftDuration = true
								return false
							}
						}

						return true
					})

					if !hasBitshiftDuration {
						return true
					}

					pos := ctx.Fset.Position(forStmt.Pos())

					f, err := finding.NewBuilder(
						"P007", toolName,
						"Manual retry loop with bitshift backoff — corrupts time.Duration "+
							"(shifts nanosecond representation), use retry.Do",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Replace with retry.Do(ctx, retry.Config{Backoff: retry.Exponential(...)}, fn)").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)
					return true
				})
			}

			return findings, nil
		},
	)
}

func mentionsDuration(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		pkg, ok := e.X.(*ast.Ident)
		return ok && pkg.Name == "time" && e.Sel.Name == "Duration"
	case *ast.Ident:
		name := e.Name
		return name == "backoff" || name == "baseBackoff" || name == "delay" ||
			name == "wait" || name == "interval"
	case *ast.BinaryExpr:
		return mentionsDuration(e.X) || mentionsDuration(e.Y)
	case *ast.ParenExpr:
		return mentionsDuration(e.X)
	}

	return false
}
