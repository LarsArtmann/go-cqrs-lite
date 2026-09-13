package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F026 detects CQRS projects that use metaengine.NewReader but never call
// metaengine.WithPrefetch. Without prefetch, every Scan/Get call hits the
// underlying store individually. WithPrefetch batches reads to reduce
// round-trips, especially important for SQL-backed engines.
//
// Fires only when the project imports metaengine (HasMetaengine).
//
//nolint:ireturn // factory returns public interface
func NewF026Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F026-no-metaengine-prefetch",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !importsPathIn(sc.files, "go-cqrs-lite/metaengine") &&
					!sc.profile.HasMetaengine {
					continue
				}

				if !hasCallIn(sc.files, "metaengine", "NewReader") {
					continue
				}

				if hasCallIn(sc.files, "metaengine", "WithPrefetch") {
					continue
				}

				pos, ok := firstNewReaderPosIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F026",
					"metaengine.NewReader used but WithPrefetch never called — "+
						"every Scan/Get hits the underlying store individually",
					"Pass metaengine.WithPrefetch(n) to reader.Scan/Get calls to "+
						"batch reads and reduce round-trips, especially for SQL engines. "+
						"Example: reader.Scan(ctx, metaengine.WithPrefetch(100), "+
						"metaengine.WithLimit(50))",
					pos, finding.ConfidenceLow,
				)...)
			}

			return out, nil
		},
	)
}

func firstNewReaderPosIn(
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
			if !ok {
				return true
			}

			if sel.Sel.Name != "NewReader" {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "metaengine" {
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
