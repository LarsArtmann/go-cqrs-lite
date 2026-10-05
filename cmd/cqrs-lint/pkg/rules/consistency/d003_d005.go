package consistency

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// D003: Inconsistent logging library.
// Detects projects mixing log/slog, log, zap, zerolog, etc.
//
//nolint:ireturn // factory returns public interface
func NewD003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D003-inconsistent-logging-library",
		func(_ context.Context) ([]finding.Finding, error) {
			loggingImports := make(map[string]bool)
			firstFile := ""
			firstLine := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, imp := range gf.AST.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					lib := ""

					switch {
					case strings.Contains(path, "/log/slog") || path == "log/slog":
						lib = "log/slog"
					case strings.Contains(path, "charm.land/log"):
						lib = "charm.land/log"
					case strings.Contains(path, "go.uber.org/zap"):
						lib = "go.uber.org/zap"
					case strings.Contains(path, "github.com/rs/zerolog"):
						lib = "zerolog"
					}

					if lib == "" {
						continue
					}

					loggingImports[lib] = true

					if firstFile == "" {
						pos := ctx.Fset.Position(imp.Pos())
						firstFile = pos.Filename
						firstLine = pos.Line
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

			f, err := findingTemplate.Builder(
				"D003",
				fmt.Sprintf(
					"Project mixes %d logging libraries: %s — standardize on one",
					len(libs),
					strings.Join(libs, ", "),
				),
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(firstFile), firstLine, 1),
			).
				WithCategory(finding.CategoryNaming).
				WithConfidence(finding.ConfidenceHigh).
				WithSuggestion("Standardize on log/slog (Go stdlib) for structured logging consistency").
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// D005: Stale documentation version.
// Detects README or docs referencing a different go-cqrs-lite version than go.mod.
//
//nolint:ireturn // factory returns public interface
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

			// Per-module pins: go-cqrs-lite tags are per-module, so docs may
			// legitimately reference several different current versions
			// ("event v4.12.0, decider v4.7.0"). A doc token is judged
			// against the module it names, falling back to any-pin matching.
			pins := readGoModCQRSVersionSet(ctx.ProjectRoot + "/go.mod")

			var findings []finding.Finding

			docFiles := []string{"README.md", "AGENTS.md", "MIGRATION.md"}

			for _, docFile := range docFiles {
				path := ctx.ProjectRoot + "/" + docFile

				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				docVersion := extractCQRSVersion(string(content), modVersion, pins)
				if docVersion == "" || docVersion == modVersion {
					continue
				}

				f, err := findingTemplate.Builder(
					"D005",
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
