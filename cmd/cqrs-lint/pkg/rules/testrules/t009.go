package testrules

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// T009: a test file hand-boots a real system via system.New instead of using
// the systemscenario harness. systemscenario (ADR-0153) boots the SAME
// DomainConfig/DeploymentConfig the production binary uses — journal,
// projection host, bus, frozen clock — and asserts on observable outcomes;
// each hand-rolled boot re-invents that wiring and drifts from the
// composition root. This is the consumer-side twin of the library's own
// check-systemscenario-adoption gate (which owns self-lint through its
// reasoned allowlist); consumers with a legitimate lower-level boot suppress
// with //cqrs-lint:ignore(T009) <reason>.
//
// Fires ONLY when a test file actually calls system.New — a project whose
// tests never boot a system owes no harness. Any systemscenario import
// (System for fresh boots, Adopt for wrapper facades that own system.New)
// satisfies the rule.
//
//nolint:ireturn // factory returns public interface
func NewT009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T009-hand-booted-system-in-tests",
		func(_ context.Context) ([]finding.Finding, error) {
			gf, call, ok := firstTestFilePkgFuncCall(ctx, "system", "New")
			if !ok {
				return nil, nil
			}

			if anyFileImports(ctx, "go-cqrs-lite/systemscenario") {
				return nil, nil
			}

			pos := ctx.Fset.Position(call.Pos())

			f, err := finding.NewBuilder(
				"T009",
				toolName,
				"Test boots a real system via system.New instead of the systemscenario harness — "+
					"hand-rolled boots drift from the composition root",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(gf.Path), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryTesting).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion(
					"Use systemscenario.System(t, ctx, domain, deploy) to boot the same "+
						"DomainConfig/DeploymentConfig as production with a frozen clock — Given/When/Then, "+
						"journal diffs, query awaits, saga capture (ADR-0153); systemscenario.Adopt wraps a "+
						"system you must boot yourself. Suppress with //cqrs-lint:ignore(T009) <reason> when a "+
						"lower-level boot is deliberate.",
				).
				WithSnippet(ctx.SourceLine(gf.Path, pos.Line)).
				Build()
			if err != nil {
				return nil, nil
			}

			return []finding.Finding{f}, nil
		},
	)
}

// firstTestFilePkgFuncCall returns the first test-file call of the form
// pkgName.funcName(...) — e.g. system.New(...) — anchored at the call site
// where the migration to the harness lands.
func firstTestFilePkgFuncCall(
	ctx *analyzer.AnalysisContext,
	pkgName, funcName string,
) (*analyzer.GoFile, *ast.CallExpr, bool) {
	for _, gf := range ctx.GoFiles {
		if !gf.IsTest || gf.AST == nil {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok || sel.Sel.Name != funcName {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != pkgName {
				return true
			}

			hit = call

			return false
		})

		if hit != nil {
			return gf, hit, true
		}
	}

	return nil, nil, false
}
