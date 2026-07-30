package version

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// V004: Vendored copy of go-cqrs-lite in a third_party directory.
// Detects Go files inside a third_party/ directory that import go-cqrs-lite
// modules. Unlike vendor/ (which A019 handles), third_party/ copies bypass
// go.mod entirely and silently miss bug fixes and security patches.
//
//nolint:ireturn // factory returns public interface
func NewV004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V004-vendored-third-party",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding
			seen := make(map[string]bool)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !isInThirdParty(gf.Path) {
					continue
				}

				hasCQRSImport := false
				for _, imp := range gf.AST.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					if analyzer.IsCQRSModulePath(path) || strings.Contains(path, "eventtest") {
						hasCQRSImport = true
						break
					}
				}

				if !hasCQRSImport {
					continue
				}

				if seen[gf.Path] {
					continue
				}
				seen[gf.Path] = true

				f, err := finding.NewBuilder(
					"V004", toolName,
					"Vendored go-cqrs-lite code in third_party/ — bypasses go.mod, "+
						"misses bug fixes and security patches",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(gf.Path), 1, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Remove the third_party copy and use a proper go.mod dependency").
					WithSnippet(ctx.SourceLine(gf.Path, 1)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}
