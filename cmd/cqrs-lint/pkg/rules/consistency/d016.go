package consistency

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// D016: Event payload struct with too many fields (>20).
// Large payload structs are hard to evolve, serialize, and reason about.
// They often signal that an event carries too much information — consider
// splitting into multiple smaller events or using a reference ID instead of
// embedding the full entity state.
//
//nolint:ireturn // factory returns public interface
func NewD016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D016-payload-too-many-fields",
		func(_ context.Context) ([]finding.Finding, error) {
			const maxFields = 20

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok {
						continue
					}

					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						structType, ok := typeSpec.Type.(*ast.StructType)
						if !ok {
							continue
						}

						structName := typeSpec.Name.Name
						if !lintutil.IsEventPayloadName(structName) {
							continue
						}

						count := countFields(structType.Fields)
						if count <= maxFields {
							continue
						}

						pos := ctx.Fset.Position(typeSpec.Pos())

						f, err := finding.NewBuilder(
							"D016", toolName,
							fmt.Sprintf(
								"Event payload %s has %d fields (max %d) — consider splitting into smaller events or using reference IDs",
								structName, count, maxFields,
							),
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Split large payloads into smaller, focused events or reference entity state by ID").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)
					}
				}
			}

			return findings, nil
		},
	)
}

// countFields counts the total number of fields in a struct, including
// fields in anonymous nested structs.
func countFields(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}

	count := 0

	for _, field := range fields.List {
		if field.Names == nil {
			// Anonymous/embedded field — count as 1.
			count++
			continue
		}

		count += len(field.Names)
	}

	return count
}
