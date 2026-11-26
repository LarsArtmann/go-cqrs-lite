package architecture

import (
	"context"
	"go/ast"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// E016: Missing health checks in server-mode projects.
// Detects modules that use stack.Bundle or have HTTP servers but don't
// call HealthCheck. Kubernetes liveness/readiness probes need this.
//
// Evaluated per module (grouped like E009): a server in one module is not
// silenced by a /healthz route, a cqrs-htmx import or a ServerLocal profile
// belonging to a different module.
//
//nolint:ireturn // factory returns public interface
func NewE016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E016-missing-health-checks",
		func(_ context.Context) ([]finding.Finding, error) {
			modules := map[string]*e016Module{}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				state := e016ModuleFor(ctx, modules, gf)
				e016ScanFile(ctx, gf, state)

				// cqrs-htmx provides an integrated server with built-in health
				// checking: its module has probes, its neighbours do not.
				if fileImportsPath(gf, "cqrs-htmx") {
					state.hasHealthCheck = true
				}
			}

			var findings []finding.Finding

			for _, dir := range slices.Sorted(maps.Keys(modules)) {
				state := modules[dir]
				// CLI tools with embedded dashboards don't need Kubernetes
				// probes: the module's OWN profile decides.
				if !state.hasServer || state.hasHealthCheck ||
					ctx.ProfileForFile(state.trigger.Filename).ServerLocal {
					continue
				}

				f, err := e016Finding(ctx, state.trigger)
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// e016Module is one module's server/health state. trigger is the first
// Bundle/server call of that module and anchors its finding.
type e016Module struct {
	trigger        token.Position
	hasServer      bool
	hasHealthCheck bool
}

// e016ModuleFor returns the state of gf's module, creating it on first sight.
// ModuleDir is empty on hand-built (unit-test) contexts; the file's directory
// is the equivalent key there.
func e016ModuleFor(
	ctx *analyzer.AnalysisContext,
	modules map[string]*e016Module,
	gf *analyzer.GoFile,
) *e016Module {
	dir := gf.ModuleDir
	if dir == "" {
		dir = filepath.Dir(ctx.Fset.Position(gf.AST.Package).Filename)
	}

	if modules[dir] == nil {
		modules[dir] = &e016Module{}
	}

	return modules[dir]
}

// e016ScanFile records the server triggers and health signals of one file.
func e016ScanFile(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile, state *e016Module) {
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

		isBundle := strings.Contains(callStr, "Bundle") || strings.Contains(callStr, "stack.New")
		isServer := sel.Sel.Name == "ListenAndServe" || sel.Sel.Name == "ListenAndServeTLS" ||
			strings.Contains(callStr, "http.Server")

		if isBundle || isServer {
			if !state.hasServer {
				state.trigger = ctx.Fset.Position(call.Pos())
			}

			state.hasServer = true
		}

		if sel.Sel.Name == "HealthCheck" || strings.Contains(callStr, "HealthCheck") {
			state.hasHealthCheck = true
		}

		return true
	})

	if isHealthRoute(gf.AST) {
		state.hasHealthCheck = true
	}
}

// isHealthRoute reports whether the file registers a health endpoint: only a
// string literal that is the first argument to a routing call (HandleFunc,
// Handle, Mount, Get, Post, ...) counts, which avoids false positives from
// health-related strings in comments, descriptions or plain assignments.
func isHealthRoute(file *ast.File) bool {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok || !routingMethods[sel.Sel.Name] || len(call.Args) == 0 {
			return true
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if ok && lit.Kind == token.STRING && isHealthEndpoint(strings.Trim(lit.Value, `"`)) {
			found = true
		}

		return true
	})

	return found
}

func e016Finding(ctx *analyzer.AnalysisContext, pos token.Position) (finding.Finding, error) {
	return findingTemplate.Builder(
		"E016",
		"Server-mode project without HealthCheck — Kubernetes probes need stack.Bundle.HealthCheck()",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceMedium).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion("Add bundle.HealthCheck(ctx) to your /healthz or /readyz endpoint").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
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
	"HandleFunc":    true,
	"Handle":        true,
	"Mount":         true,
	"Get":           true,
	"Post":          true,
	"Put":           true,
	"Patch":         true,
	"Delete":        true,
	"Any":           true,
	"Group":         true,
	"Route":         true,
	"Path":          true,
	"PathPrefix":    true,
	"Method":        true,
	"AddRoute":      true,
	"RegisterRoute": true,
}
