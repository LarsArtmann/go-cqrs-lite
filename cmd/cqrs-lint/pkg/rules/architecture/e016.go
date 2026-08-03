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

			// CLI tools with embedded dashboards don't need Kubernetes probes.
			if ctx.FeatureProfile.ServerLocal {
				return nil, nil
			}

			hasBundle := false
			hasServer := false
			hasHealthCheck := false
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

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					callStr := analyzer.ExprString(call.Fun)

					if strings.Contains(callStr, "Bundle") ||
						strings.Contains(callStr, "stack.New") {
						if !hasBundle && !hasServer {
							triggerPos = ctx.Fset.Position(call.Pos())
						}
						hasBundle = true
					}

					if sel.Sel.Name == "ListenAndServe" || sel.Sel.Name == "ListenAndServeTLS" ||
						strings.Contains(callStr, "http.Server") {
						if !hasBundle && !hasServer {
							triggerPos = ctx.Fset.Position(call.Pos())
						}
						hasServer = true
					}

					if sel.Sel.Name == "HealthCheck" || strings.Contains(callStr, "HealthCheck") {
						hasHealthCheck = true
					}

					return true
				})

				// Recognize health endpoint route registrations: only match string
				// literals that are the first argument to a routing function call
				// (HandleFunc, Handle, Mount, Get, Post, etc.). This avoids false
				// positives from health-related strings in comments, descriptions,
				// or non-routing variable assignments.
				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if !routingMethods[sel.Sel.Name] {
						return true
					}

					if len(call.Args) == 0 {
						return true
					}

					lit, ok := call.Args[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}

					if isHealthEndpoint(strings.Trim(lit.Value, `"`)) {
						hasHealthCheck = true
					}

					return true
				})
			}

			if (hasBundle || hasServer) && !hasHealthCheck {
				f, err := finding.NewBuilder(
					"E016", toolName,
					"Server-mode project without HealthCheck — Kubernetes probes need stack.Bundle.HealthCheck()",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(triggerPos.Filename), triggerPos.Line, triggerPos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Add bundle.HealthCheck(ctx) to your /healthz or /readyz endpoint").
					WithSnippet(ctx.SourceLine(triggerPos.Filename, triggerPos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// healthEndpoints are URL paths that signal a health/readiness probe.
// When any of these appears as a string literal in non-test code, the
// project has a health endpoint and E016 should not fire.
//
//nolint:gochecknoglobals // read-only lookup set
var healthEndpoints = map[string]bool{
	"/health":  true,
	"/healthz": true,
	"/ready":   true,
	"/readyz":  true,
	"/livez":   true,
}

func isHealthEndpoint(path string) bool {
	return healthEndpoints[path]
}

// routingMethods are HTTP-router method names whose first string argument is
// a route path. Only these methods are checked for health-endpoint string
// literals — this narrows the scan from "any string literal anywhere" to
// "route registration calls."
//
//nolint:gochecknoglobals // read-only lookup set
var routingMethods = map[string]bool{
	"HandleFunc":     true,
	"Handle":         true,
	"Mount":          true,
	"Get":            true,
	"Post":           true,
	"Put":            true,
	"Patch":          true,
	"Delete":         true,
	"Any":            true,
	"Group":          true,
	"Route":          true,
	"Path":           true,
	"PathPrefix":     true,
	"Method":         true,
	"AddRoute":       true,
	"RegisterRoute":  true,
}
