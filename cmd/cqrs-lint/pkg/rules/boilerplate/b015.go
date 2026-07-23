package boilerplate

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B015: Missing test utilities.
// Detects projects with test files but no testutil/eventtest imports.
//
//nolint:ireturn // factory returns public interface
func NewB015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B015-missing-test-utilities",
		func(_ context.Context) ([]finding.Finding, error) {
			hasTestFiles := false
			hasTestUtils := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					hasTestFiles = true
				}

				for _, pkg := range ctx.Packages {
					for _, imp := range pkg.Imports {
						if imp == nil {
							continue
						}

						if strings.Contains(imp.PkgPath, "testutil") ||
							strings.Contains(imp.PkgPath, "eventtest") ||
							strings.Contains(imp.PkgPath, "querytest") ||
							strings.Contains(imp.PkgPath, "scenario") {
							hasTestUtils = true
						}
					}
				}
			}

			if !hasTestFiles || hasTestUtils {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"B015", toolName,
				"Project has test files but no testutil/eventtest imports — manual test setup is error-prone",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Use event/v4/eventtest for fake stores/buses, or scenario for BDD-style decider testing").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
