package architecture

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// E016: Missing health checks in server-mode projects.
// Detects projects that use stack.Bundle or have HTTP servers but don't
// call HealthCheck. Kubernetes liveness/readiness probes need this.
//
//nolint:ireturn // factory returns public interface
func NewE016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E016-missing-health-checks",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasBundle := false
			hasServer := false
			hasHealthCheck := false

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

					callStr := analyzer.ExprString(call.Fun)

					if strings.Contains(callStr, "Bundle") || strings.Contains(callStr, "stack.New") {
						hasBundle = true
					}

					if sel.Sel.Name == "ListenAndServe" || sel.Sel.Name == "ListenAndServeTLS" ||
						strings.Contains(callStr, "http.Server") {
						hasServer = true
					}

					if sel.Sel.Name == "HealthCheck" || strings.Contains(callStr, "HealthCheck") {
						hasHealthCheck = true
					}

					return true
				})
			}

			if (hasBundle || hasServer) && !hasHealthCheck {
				pos := finding.Pos("project", 1, 1)

				f, err := finding.NewBuilder(
					"E016", toolName,
					"Server-mode project without HealthCheck — Kubernetes probes need stack.Bundle.HealthCheck()",
					finding.SeverityWarning,
					pos,
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Add bundle.HealthCheck(ctx) to your /healthz or /readyz endpoint").
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}
