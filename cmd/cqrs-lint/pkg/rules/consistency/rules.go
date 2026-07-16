// Package consistency implements consistency-checking rules.
package consistency

import (
	"context"
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const toolName finding.ToolName = "cqrs-lint"

// D001: Inconsistent event naming.
// Detects events with inconsistent naming conventions (mix of PascalCase, snake_case, camelCase).
func NewD001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D001-inconsistent-event-naming",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasDotNotation := false
			hasNoDotNotation := false
			firstFile := ""
			firstLine := 0

			for eventType, emission := range ctx.Registry.EventTypesEmitted {
				if strings.Contains(eventType, ".") {
					hasDotNotation = true
				} else {
					hasNoDotNotation = true
				}

				if firstFile == "" && emission.File != "" {
					firstFile = emission.File
					firstLine = emission.Line
				}
			}

			if hasDotNotation && hasNoDotNotation {
				pos := finding.Pos(finding.FilePath(firstFile), firstLine, 1)
				if firstFile == "" {
					pos = finding.Pos(
						finding.FilePath(filepath.Join(ctx.ProjectRoot, "go.mod")),
						1,
						1,
					)
				}

				f, err := finding.NewBuilder(
					"D001", toolName,
					"Inconsistent event type naming — some use dot notation (user.created), others don't (UserCreated)",
					finding.SeverityInfo,
					pos,
				).
					WithCategory(finding.CategoryStyle).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Pick one convention: dot notation (domain.event) or PascalCase (DomainEvent) and use it consistently").
					WithSnippet(ctx.SourceLine(firstFile, firstLine)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// D002: Inconsistent JSON casing.
// Detects mixing camelCase and snake_case in JSON struct tags within the same package.
func NewD002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D002-inconsistent-json-casing",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				hasCamel := false
				hasSnake := false

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					st, ok := n.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					for _, field := range st.Fields.List {
						if field.Tag == nil {
							continue
						}

						tag := field.Tag.Value

						jsonTag := analyzer.ExtractJSONTag(tag)
						if jsonTag == "" || jsonTag == "-" {
							continue
						}

						if strings.Contains(jsonTag, "_") {
							hasSnake = true
						} else if hasLower(jsonTag) {
							hasCamel = true
						}
					}

					return true
				})

				if hasCamel && hasSnake {
					pos := ctx.Fset.Position(gf.AST.Pos())

					f, err := finding.NewBuilder(
						"D002", toolName,
						fmt.Sprintf("File %s mixes camelCase and snake_case JSON tags — pick one convention", gf.Path),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryStyle).
						WithConfidence(finding.ConfidenceLow).
						WithSuggestion("Pick one JSON key casing convention — camelCase for API types, snake_case for event payloads").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

func hasLower(s string) bool {
	if s == "" {
		return false
	}

	return s[0] >= 'a' && s[0] <= 'z'
}
