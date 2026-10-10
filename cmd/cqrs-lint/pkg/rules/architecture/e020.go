package architecture

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// systemModulePath is the exact import path of the composition root. An
// exact match (not Contains) keeps siblings like systemscenario/v4 and
// system/integration from tripping the qualifier check in typed mode.
const systemModulePath = "github.com/larsartmann/go-cqrs-lite/system/v4"

// E020: a test file boots the composition root by hand.
// `system.New` in a _test.go file is usually a hand-rolled scenario boot —
// bespoke setup/teardown, ad-hoc journal reads, sleeping await loops — that
// the systemscenario harness already provides as tested infrastructure
// (Given/When/Then over a real boot, journal capture, poll-based read-model
// assertions, a vacuous-pass guard). The advisory fires when the test file
// calls system.New WITHOUT importing systemscenario; tests that boot the
// system to hand it to `systemscenario.Adopt` import the harness and stay
// silent by construction. Production files are never flagged — booting the
// composition root is exactly what production code is supposed to do.
//
// Advisory by design: engine-coverage suites (systemtest-style) deliberately
// boot raw engines under system.New, and the harness would add nothing for
// them; the finding's suggestion names Adopt/System as the default, not a
// mandate.
//
//nolint:ireturn // factory returns public interface
func NewE020Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E020-handrolled-system-boot-in-test",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if !gf.IsTest || fileAdoptsSystemscenario(gf.AST) {
					continue
				}

				for _, call := range systemNewCalls(gf) {
					if f, err := e020Finding(ctx, gf, call); err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// fileAdoptsSystemscenario reports whether the file imports the
// systemscenario harness (under any local alias).
func fileAdoptsSystemscenario(file *ast.File) bool {
	if file == nil {
		return false
	}

	for _, imp := range file.Imports {
		if strings.Contains(imp.Path.Value, "go-cqrs-lite/systemscenario") {
			return true
		}
	}

	return false
}

// systemNewCalls returns the system.New call sites in the file. The
// qualifier resolves through type info when available (aliased imports
// match by exact import path); the import-table fallback covers
// syntax-only loads, mapping the qualifier to its import's local name
// (alias, or last path segment with the major-version suffix stripped).
func systemNewCalls(gf *analyzer.GoFile) []*ast.CallExpr {
	var calls []*ast.CallExpr

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "New" {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		if qualifierIsSystem(gf, ident) {
			calls = append(calls, call)
		}

		return true
	})

	return calls
}

// qualifierIsSystem reports whether ident refers to the system module.
// Typed resolution wins (exact import path, shadowing-aware); the
// import-table fallback keeps syntax-only loads honest for aliases.
func qualifierIsSystem(gf *analyzer.GoFile, ident *ast.Ident) bool {
	if path, resolved := analyzer.ResolveQualifierTyped(gf, ident); resolved {
		return path == systemModulePath
	}

	if gf == nil || gf.AST == nil {
		return false
	}

	for _, imp := range gf.AST.Imports {
		if importLocalName(imp) == ident.Name {
			return strings.Trim(imp.Path.Value, `"`) == systemModulePath
		}
	}

	return false
}

// importLocalName derives the local name an import binds: the explicit
// alias, or the last path segment with a major-version suffix ("/v4")
// stripped (the package name convention for versioned module paths).
func importLocalName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}

	path := strings.Trim(imp.Path.Value, `"`)
	if idx := strings.LastIndex(path, "/"); idx >= 0 && isVersionSegment(path[idx+1:]) {
		path = path[:idx]
	}

	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}

	return path
}

// isVersionSegment reports whether s looks like a module major-version
// suffix: "v" followed by digits only.
func isVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}

	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// e020Finding builds the finding for one hand-rolled boot, anchored at the
// system.New call site.
func e020Finding(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	call *ast.CallExpr,
) (finding.Finding, error) {
	pos := ctx.Fset.Position(call.Pos())

	return findingTemplate.Builder(
		"E020",
		"Test file boots system.New by hand instead of adopting the systemscenario harness",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(gf.Path), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion(
			"Use systemscenario.System(t, ctx, domain, deploy) for scenario boots " +
				"(or systemscenario.Adopt over this boot) — Given/When/Then, journal capture, " +
				"poll-based read-model assertions, and a vacuous-pass guard come with it; " +
				"engine-coverage suites that genuinely need the raw boot may dismiss this advisory",
		).
		WithSnippet(ctx.SourceLine(gf.Path, pos.Line)).
		Build()
}
