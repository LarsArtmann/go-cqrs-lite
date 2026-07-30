package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C029: QueryIdempotency with nil keyExtractor panics at runtime.
// Unlike CommandIdempotency and EventIdempotency which default to
// cmd.ID().String() / evt.ID().String(), QueryIdempotency has no built-in
// identity and requires an explicit keyExtractor. Passing nil panics.
//
//nolint:ireturn // factory returns public interface
func NewC029Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C029-queryidempotency-nil-keyextractor",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

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
					if !ok || sel.Sel.Name != "QueryIdempotency" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "middleware" {
						return true
					}

					// keyExtractor is the 3rd positional arg (index 2).
					if len(call.Args) < 3 {
						return true
					}

					ident, ok := call.Args[2].(*ast.Ident)
					if !ok || ident.Name != "nil" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C029", toolName,
						"QueryIdempotency called with nil keyExtractor — "+
							"this panics at runtime (queries have no default identity)",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Provide a keyExtractor function, e.g. " +
							"middleware.QueryIdempotency(store, ttl, " +
							"func(q query.Query) string { return string(q.Type()) })").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}
