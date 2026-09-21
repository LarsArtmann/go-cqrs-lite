package adoption

import (
	"context"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F031 detects metaengine reader Scan calls that pass no WithLimit option.
// Without WithLimit a Scan silently truncates at 100 rows — a correctness
// hazard, not a performance one (consumers reading page 101 never know rows
// are missing; see G-T14). The nudge coaches the three sanctioned paths:
// an explicit WithLimit per scan, WithLimit(0) for intentionally unbounded
// reads, or the operator's store-wide WithDefaultLimit ceiling.
//
// Fires only when the project imports metaengine (HasMetaengine). Skipped
// entirely when the project sets metaengine.WithDefaultLimit — the operator
// has already bounded the store.
//
//nolint:ireturn // factory returns public interface
func NewF031Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F031-scan-without-limit",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !importsPathIn(sc.files, "go-cqrs-lite/metaengine") &&
					!sc.profile.HasMetaengine {
					continue
				}

				if hasCallIn(sc.files, "metaengine", "WithDefaultLimit") {
					continue
				}

				pos, ok := firstScanWithoutLimitPosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleWarningFinding(
					ctx,
					"F031",
					"Scan without WithLimit silently truncates at 100 rows — "+
						"larger collections return wrong (partial) results",
					"Pass metaengine.WithLimit(n) with the page size you want "+
						"(WithLimit(0) for intentionally unbounded), or set the "+
						"store-wide operator ceiling metaengine.WithDefaultLimit(n) "+
						"at Plan time",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

// firstScanWithoutLimitPosIn returns the position of the first non-test
// method call named Scan whose arguments carry no WithLimit option. The
// check is syntactic (no type info): a `X.Scan(...)` without WithLimit in
// a metaengine-importing project is the reader scan with overwhelming
// probability, hence the low catalog confidence.
func firstScanWithoutLimitPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok || sel.Sel.Name != "Scan" {
				return true
			}

			if slices.ContainsFunc(call.Args, callsWithLimit) {
				return true
			}

			hit = call

			return false
		})

		if hit != nil {
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// callsWithLimit reports whether the expression references a WithLimit
// identifier (metaengine.WithLimit or a locally aliased option value).
func callsWithLimit(expr ast.Expr) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		switch e := n.(type) {
		case *ast.SelectorExpr:
			if strings.Contains(e.Sel.Name, "WithLimit") {
				found = true

				return false
			}
		case *ast.Ident:
			if strings.Contains(e.Name, "WithLimit") {
				found = true

				return false
			}
		}

		return true
	})

	return found
}
