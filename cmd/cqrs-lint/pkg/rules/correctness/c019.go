package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects multiple decider.NewRepository calls with the same state type.
// Each instance has its own singleflight group and state cache, so duplicate
// instances waste memory and defeat load coalescing.
//
//nolint:ireturn // factory returns public interface
func NewC019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C019-multiple-repos-same-aggregate",
		func(_ context.Context) ([]finding.Finding, error) {
			// Map state-type → positions where NewRepository[StateType] is called.
			type repoCall struct {
				typeParam string
				pos       tokenPos
			}

			var calls []repoCall

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

					if sel.Sel.Name != "NewRepository" && sel.Sel.Name != "NewTypedRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					typeParam := extractTypeParam(call.Fun)
					if typeParam == "" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					calls = append(calls, repoCall{
						typeParam: typeParam,
						pos:       tokenPos{file: pos.Filename, line: pos.Line, col: pos.Column},
					})

					return true
				})
			}

			// Group by type param.
			byType := make(map[string][]tokenPos)
			for _, c := range calls {
				byType[c.typeParam] = append(byType[c.typeParam], c.pos)
			}

			var findings []finding.Finding

			for typeParam, positions := range byType {
				if len(positions) <= 1 {
					continue
				}

				// Fire on the second+ calls.
				for _, p := range positions[1:] {
					f, err := finding.NewBuilder(
						"C019", toolName,
						"Multiple Repository instances for "+typeParam+
							" — wastes singleflight/cache, share one instance",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(p.file), p.line, p.col),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Create one Repository[" + typeParam +
							"] and share it across handlers").
						WithSnippet(ctx.SourceLine(p.file, p.line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

type tokenPos struct {
	file string
	line int
	col  int
}

// extractTypeParam extracts the type parameter from a generic call like
// decider.NewRepository[StateType](...).
func extractTypeParam(fun ast.Expr) string {
	switch e := fun.(type) {
	case *ast.IndexExpr:
		return analyzer.ExprString(e.Index)
	case *ast.IndexListExpr:
		if len(e.Indices) > 0 {
			return analyzer.ExprString(e.Indices[0])
		}
	}

	return ""
}
