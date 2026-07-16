package consistency

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// D003: Inconsistent logging library.
// Detects projects mixing log/slog, log, zap, zerolog, etc.
func NewD003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D003-inconsistent-logging-library",
		func(_ context.Context) ([]finding.Finding, error) {
			loggingImports := make(map[string]bool)

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					path := imp.PkgPath
					if strings.Contains(path, "/log/slog") || path == "log/slog" {
						loggingImports["log/slog"] = true
					}

					if strings.Contains(path, "charm.land/log") {
						loggingImports["charm.land/log"] = true
					}

					if strings.Contains(path, "go.uber.org/zap") {
						loggingImports["go.uber.org/zap"] = true
					}

					if strings.Contains(path, "github.com/rs/zerolog") {
						loggingImports["zerolog"] = true
					}
				}
			}

			if len(loggingImports) <= 1 {
				return nil, nil
			}

			libs := make([]string, 0, len(loggingImports))
			for k := range loggingImports {
				libs = append(libs, k)
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"D003",
				toolName,
				fmt.Sprintf(
					"Project mixes %d logging libraries: %s — standardize on one",
					len(libs),
					strings.Join(libs, ", "),
				),
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryNaming).
				WithConfidence(finding.ConfidenceHigh).
				WithSuggestion("Standardize on log/slog (Go stdlib) for structured logging consistency").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// D004: Inconsistent JSON key casing.
// Detects struct fields with mixed camelCase and snake_case JSON tags.
func NewD004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D004-inconsistent-json-key-casing",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			camelCount := 0
			snakeCount := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

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

						jsonTag := extractJSONTag(tag)
						if jsonTag == "" || jsonTag == "-" {
							continue
						}

						if strings.Contains(jsonTag, "_") {
							snakeCount++
						} else if jsonTag[0] >= 'a' && jsonTag[0] <= 'z' {
							camelCount++
						}
					}

					return true
				})
			}

			if camelCount > 0 && snakeCount > 0 {
				f, err := finding.NewBuilder(
					"D004", toolName,
					fmt.Sprintf("Mixed JSON key casing: %d camelCase, %d snake_case — pick one convention", camelCount, snakeCount),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
				).
					WithCategory(finding.CategoryNaming).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Standardize on snake_case for JSON event payloads (conventional for event stores and APIs)").
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// D005: Stale documentation version.
// Detects README or docs referencing a different go-cqrs-lite version than go.mod.
func NewD005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D005-stale-documentation-version",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.ProjectRoot == "" {
				return nil, nil
			}

			modVersion := readGoModCQRSVersion(ctx.ProjectRoot + "/go.mod")
			if modVersion == "" {
				return nil, nil
			}

			var findings []finding.Finding

			docFiles := []string{"README.md", "AGENTS.md", "MIGRATION.md"}

			for _, docFile := range docFiles {
				path := ctx.ProjectRoot + "/" + docFile

				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				docVersion := extractCQRSVersion(string(content), modVersion)
				if docVersion == "" || docVersion == modVersion {
					continue
				}

				f, err := finding.NewBuilder(
					"D005",
					toolName,
					fmt.Sprintf(
						"%s references go-cqrs-lite %s but go.mod has %s",
						docFile,
						docVersion,
						modVersion,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(path), 1, 1),
				).
					WithCategory(finding.CategoryNaming).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Update documentation to match the version in go.mod").
					WithSnippet(ctx.SourceLine(path, 1)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

func readGoModCQRSVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		if !strings.Contains(line, "go-cqrs-lite") {
			continue
		}

		if strings.Contains(line, "replace") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		return parts[len(parts)-1]
	}

	return ""
}

func extractCQRSVersion(content, modVersion string) string {
	versions := []string{}

	for line := range strings.SplitSeq(content, "\n") {
		if !strings.Contains(strings.ToLower(line), "go-cqrs-lite") {
			continue
		}

		for field := range strings.FieldsSeq(line) {
			if strings.HasPrefix(field, "v") && len(field) > 2 {
				versions = append(versions, field)
			}
		}
	}

	if len(versions) == 0 {
		return modVersion
	}

	return versions[0]
}
