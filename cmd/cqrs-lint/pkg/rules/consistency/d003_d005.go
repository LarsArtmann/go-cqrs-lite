package consistency

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
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

			f, err := finding.NewBuilder(
				"D003",
				toolName,
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

		// Skip markdown headings (ADR titles, section headers) — these contain
		// historical version references like "ADR-0044: Migrate from v3 to v4"
		// that describe past migrations, not current version claims.
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		for field := range strings.FieldsSeq(line) {
			if !looksLikeVersionToken(field) {
				continue
			}

			// Skip migration arrows like "v2→v3" — these describe historical
			// migrations, not current version claims.
			if strings.Contains(field, "→") || strings.Contains(field, "->") {
				continue
			}

			versions = append(versions, field)
		}
	}

	if len(versions) == 0 {
		return modVersion
	}

	docVersion := versions[0]

	// Wildcard compatibility: "v4.0.x" matches any "v4.0.N" in go.mod.
	if isVersionCompatible(docVersion, modVersion) {
		return modVersion
	}

	return docVersion
}

// isVersionCompatible checks whether a doc version reference is compatible
// with the go.mod version. This handles:
//   - Wildcards: "v4.0.x" matches "v4.0.0", "v4.0.1", etc.
//   - Major.minor only: "v4.0" matches "v4.0.0"
func isVersionCompatible(docVersion, modVersion string) bool {
	docParts := parseVersionParts(docVersion)
	modParts := parseVersionParts(modVersion)

	if len(docParts) == 0 || len(modParts) == 0 {
		return false
	}

	for i := range docParts {
		if i >= len(modParts) {
			break
		}

		// Wildcard "x" matches any number
		if docParts[i] == "x" || docParts[i] == "X" {
			continue
		}

		if docParts[i] != modParts[i] {
			return false
		}
	}

	return true
}

// parseVersionParts splits a version string like "v4.0.1" into ["4", "0", "1"].
// Returns nil if the input doesn't look like a semantic version.
func parseVersionParts(v string) []string {
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil
	}

	// Strip trailing punctuation (e.g., "v4.2.0." from prose like "uses v4.2.0.")
	v = strings.TrimRight(v, ".,")

	parts := strings.Split(v, ".")

	// Trailing empty parts (from trailing dots) are stripped, not fatal.
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 0 || slices.Contains(parts, "") {
		return nil
	}

	return parts
}

// looksLikeVersionToken reports whether a whitespace-delimited token has the
// shape of a Go module version reference: a leading "v" + digit(s) + "." +
// digit(s), e.g. v4.0, v4.0.1, v4.0.x.
//
// This rejects prose words that merely start with "v" — "via", "version",
// "very", "vectors" — AND bare major versions like "v3"/"v4" that are
// ambiguous in prose ("v3 Migration", "v4 release"). A real version reference
// always includes at least major.minor. See feedback:
// docs/feedback/2026-07-16_DiscordSync (D005 false positive on "via go-cqrs-lite").
func looksLikeVersionToken(field string) bool {
	return versionTokenRe.MatchString(field)
}

var versionTokenRe = regexp.MustCompile(`^v\d+\.\d+`)
