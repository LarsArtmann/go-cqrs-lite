package version

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// V005: Vendored eventtest alongside go-cqrs-lite imports.
// Detects a local/vendored copy of the eventtest package used as a workaround
// for a published version that doesn't match the project's go-cqrs-lite stack
// version. This is a symptom of version mismatch that can cause subtle
// test-fixture incompatibilities.
//
//nolint:ireturn // factory returns public interface
func NewV005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V005-eventtest-vendored-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			hasRegularCQRSImport := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if isInThirdParty(gf.Path) || isInVendor(gf.Path) {
					continue
				}

				for _, imp := range gf.AST.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					if analyzer.IsCQRSModulePath(path) {
						hasRegularCQRSImport = true
						break
					}
				}

				if hasRegularCQRSImport {
					break
				}
			}

			if !hasRegularCQRSImport {
				return nil, nil
			}

			var findings []finding.Finding
			seen := make(map[string]bool)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				isNonStandard := isInThirdParty(gf.Path) || isInVendor(gf.Path)
				if !isNonStandard {
					continue
				}

				if !strings.Contains(gf.Path, "eventtest") {
					continue
				}

				if seen[gf.Path] {
					continue
				}
				seen[gf.Path] = true

				f, err := finding.NewBuilder(
					"V005", toolName,
					"Vendored eventtest package alongside go-cqrs-lite imports — "+
						"version mismatch between eventtest and the stack, remove the "+
						"vendored copy and update go-cqrs-lite to a version that ships "+
						"a compatible eventtest",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(gf.Path), 1, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Remove the vendored eventtest and ensure the published eventtest version matches your go-cqrs-lite stack").
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
