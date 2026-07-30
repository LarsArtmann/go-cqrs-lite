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
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

const toolName = lintutil.ToolName

// D001: Inconsistent event naming.
// Detects events with inconsistent naming conventions (mix of PascalCase, snake_case, camelCase).
//
//nolint:ireturn // factory returns public interface
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
// Detects individual structs that mix camelCase and snake_case JSON tags.
// A struct with `json:"firstName"` and `json:"guild_id"` will serialize
// inconsistently — this is always a bug within a single DTO/event type.
//
// Cross-struct mixing (struct A all camelCase, struct B all snake_case) is
// NOT flagged: different structs may legitimately follow different conventions
// (API types vs event payloads). The previous file-level check was the #1 noise
// source across all consumers (33x on KeyCountdown, 20x on DiscordSync) because
// it fired on legitimate cross-struct patterns and reported at line 1:1.
//
// Structs that mirror an external API (Discord, Stripe, GitHub) are excluded:
// their snake_case JSON tags are dictated by the upstream API and are not a
// style choice the consumer can change. See collectExternalAPIStructs for the
// two opt-out mechanisms (config prefix list + //cqrs-lint:external-api marker).
//
//nolint:ireturn // factory returns public interface
func NewD002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D002-inconsistent-json-casing",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				external := collectExternalAPIStructs(gf.AST, ctx.RulesConfig)

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					if external[st] {
						return true
					}

					hasCamel := false
					hasSnake := false

					for _, field := range st.Fields.List {
						if field.Tag == nil {
							continue
						}

						jsonTag := analyzer.ExtractJSONTag(field.Tag.Value)
						if jsonTag == "" || jsonTag == "-" {
							continue
						}

						if strings.Contains(jsonTag, "_") {
							hasSnake = true
						} else if isCamelCase(jsonTag) {
							hasCamel = true
						}
					}

					if hasCamel && hasSnake {
						pos := ctx.Fset.Position(ts.Pos())

						f, err := finding.NewBuilder(
							"D002", toolName,
							fmt.Sprintf("Struct %s mixes camelCase and snake_case JSON tags — pick one convention", ts.Name.Name),
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryStyle).
							WithConfidence(finding.ConfidenceLow).
							WithSuggestion("Pick one JSON key casing convention for this struct — camelCase for API types, snake_case for event payloads").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isCamelCase reports whether a JSON tag uses camelCase convention.
// A tag is camelCase when it has no underscores and contains at least one
// uppercase letter after the first character (e.g. "firstName", "guildId").
// Single-word tags like "id", "content", "name" are NEUTRAL — they don't
// indicate a casing convention either way and must not be counted as camelCase.
// This eliminates the false positive where structs like {content, guild_id}
// were flagged as "mixing camelCase and snake_case" even though "content" is
// just a single lowercase word consistent with snake_case convention.
func isCamelCase(s string) bool {
	if strings.Contains(s, "_") {
		return false
	}

	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}

	return false
}
