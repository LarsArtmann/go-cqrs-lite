package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects time.Local usage in Go source files.
//
// time.Local introduces server-local timezone dependencies that cause
// silent data corruption when events cross timezone boundaries.
// Use time.UTC (or event.Instant) instead.
func NewC014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C014-time-local-usage",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					ident, ok := sel.X.(*ast.Ident)
					if !ok || ident.Name != "time" {
						return true
					}

					if sel.Sel.Name != "Local" {
						return true
					}

					pos := ctx.Fset.Position(sel.Pos())

					f, err := finding.NewBuilder(
						"C014",
						toolName,
						fmt.Sprintf(
							"time.Local usage at %s — server-local timezone causes silent data corruption across timezones",
							pos.String(),
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion(
							"Use time.UTC instead of time.Local. " +
								"For event payload timestamps, use event.Instant. " +
								"See docs/TIMEZONE_HANDLING.md.",
						).
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
