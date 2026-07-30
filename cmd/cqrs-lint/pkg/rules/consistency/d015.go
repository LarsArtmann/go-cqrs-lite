package consistency

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D015: Nullable pointer fields in event payloads.
// `*string`, `*int` pointer fields in event payloads can cause nil-dereference
// panics on decode. Suggest value types with `omitempty` or sentinel values.
//
//nolint:ireturn // factory returns public interface
func NewD015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D015-nullable-payload-fields",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					if !lintutil.IsEventPayloadName(ts.Name.Name) &&
						!ctx.Registry.EventPayloadTypes[ts.Name.Name] {
						return true
					}

					for _, field := range st.Fields.List {
						if len(field.Names) == 0 {
							continue
						}

						starExpr, ok := field.Type.(*ast.StarExpr)
						if !ok {
							continue
						}

						// Only flag primitive-type pointers (*string, *int, etc.)
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							if !isPrimitiveType(ident.Name) {
								continue
							}
						} else {
							continue
						}

						pos := ctx.Fset.Position(field.Pos())
						fieldName := field.Names[0].Name

						f, err := finding.NewBuilder(
							"D015", toolName,
							"Event payload field "+ts.Name.Name+"."+fieldName+
								" is a pointer — nil-dereference panic risk on decode",
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Use value type with omitempty instead of pointer to avoid nil-dereference").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

func isPrimitiveType(name string) bool {
	switch name {
	case "string", "int", "int64", "int32", "float64", "float32", "bool", "uint", "uint64":
		return true
	default:
		return false
	}
}
