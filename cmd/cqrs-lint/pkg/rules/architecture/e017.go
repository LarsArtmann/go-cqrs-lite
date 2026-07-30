package architecture

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// E017: Missing graceful shutdown.
// Detects projects that use signal.Notify (indicating signal handling) but
// don't call GracefulClose or Stop. On SIGTERM, in-flight events and
// projections are killed without draining.
//
//nolint:ireturn // factory returns public interface
func NewE017Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E017-missing-graceful-shutdown",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasSignalNotify := false
			hasGracefulShutdown := false
			var triggerPos token.Position

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					callStr := analyzer.ExprString(call.Fun)

					if strings.Contains(callStr, "signal.Notify") {
						if !hasSignalNotify {
							triggerPos = ctx.Fset.Position(call.Pos())
						}
						hasSignalNotify = true
					}

					if strings.Contains(callStr, "GracefulClose") ||
						strings.Contains(callStr, ".Stop()") ||
						strings.Contains(callStr, ".Shutdown(") {
						hasGracefulShutdown = true
					}

					return true
				})
			}

			if hasSignalNotify && !hasGracefulShutdown {
				f, err := finding.NewBuilder(
					"E017", toolName,
					"signal.Notify without GracefulClose/Stop — in-flight events lost on SIGTERM",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(triggerPos.Filename), triggerPos.Line, triggerPos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Call bundle.GracefulClose(ctx) or projectionhost.Stop() on signal receipt").
					WithSnippet(ctx.SourceLine(triggerPos.Filename, triggerPos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}
