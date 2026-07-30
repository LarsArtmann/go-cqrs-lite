package consistency

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D014: Event payload struct without json tags.
// Event payload structs without `json:"..."` tags use Go field names in JSON
// encoding, producing PascalCase keys inconsistent with typical JSON
// conventions (camelCase or snake_case). This causes silent compatibility
// breaks when switching between JSON and CBOR codecs.
//
// Only fires on structs whose name ends in "Created", "Updated", "Deleted",
// "Event", or that are in the EventPayloadTypes registry — matching the
// CQRS event payload naming convention.
//
//nolint:ireturn // factory returns public interface
func NewD014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D014-event-payload-without-json-tags",
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

						if hasJSONTag(field) {
							continue
						}

						pos := ctx.Fset.Position(field.Pos())
						fieldName := field.Names[0].Name

						f, err := finding.NewBuilder(
							"D014", toolName,
							"Event payload field "+ts.Name.Name+"."+fieldName+
								" lacks json tag — Go field name used in JSON encoding",
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Add `json:\"fieldName\"` tag").
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

// hasJSONTag reports whether the field has a `json:"..."` struct tag.
func hasJSONTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}

	tag := field.Tag.Value
	return strings.Contains(tag, "json:")
}
