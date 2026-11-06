package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F026 detects CQRS projects that use metaengine.NewReader but never attach a
// PrefetchCache. Without one, every Scan page fetch hits the underlying store
// individually; a cache batches cursor-page reads to reduce round-trips,
// especially important for SQL-backed engines. Point-Get reads do not use the
// prefetch cache (typed_reader.go), so readers that only Get are not coached —
// the gate is an actual .Scan call site (nsfw-classifier feedback, 2026-10-03).
//
// Either attach form satisfies the rule: the reader-level
// `reader.WithPrefetch(cache)` (the form that has existed since early v4) or the
// per-call `metaengine.WithPrefetch(cache)` scan option (metaengine ≥ v4.21.0).
// The check is receiver-agnostic because the reader form has no package
// identifier to match on, and it resolves import aliases so
// `metaeng "…/metaengine/v4"` is seen too.
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
				importsMetaengine := importsPathIn(sc.files, "go-cqrs-lite/metaengine")
				if !importsMetaengine && !sc.profile.HasMetaengine {
					continue
				}

				localNames := importLocalNames(sc.files, "go-cqrs-lite/metaengine")

				if !hasCallToLocal(sc.files, localNames, "NewReader") {
					continue
				}

				if hasCallToNameAnywhere(sc.files, "WithPrefetch") {
					continue
				}

				if !hasCursorPaginationCall(sc.files) {
					continue
				}

				pos, ok := firstNewReaderPosIn(ctx.Fset, sc.files, localNames)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F026",
					"metaengine.NewReader used for cursor-paginated Scans but no "+
						"PrefetchCache is attached — every page fetch hits the underlying "+
						"store individually",
					"Attach a cache once: reader := metaengine.NewReader[V](store, \"q\")"+
						".WithPrefetch(metaengine.NewPrefetchCache()), or per call with "+
						"metaengine.WithPrefetch(cache) alongside WithLimit/WithCursor "+
						"(metaengine v4.21.0+). Either batches cursor-page reads, cutting "+
						"round-trips on SQL engines. Point-Get only readers need no prefetch.",
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
	localNames map[string]bool,
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
			if !ok || !localNames[ident.Name] {
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
