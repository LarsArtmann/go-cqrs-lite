package version

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects projects mixing v3 and v4 go-cqrs-lite module versions. Different
// major versions have incompatible APIs, leading to subtle runtime bugs.
//
//nolint:ireturn // factory returns public interface
func NewV001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V001-mixed-major-versions",
		func(_ context.Context) ([]finding.Finding, error) {
			hasV3 := false
			hasV4 := false
			var firstFile string
			var firstLine int

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, imp := range gf.AST.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					if !analyzer.IsCQRSModulePath(path) {
						continue
					}

					if strings.Contains(path, "/v3") {
						hasV3 = true
					}

					if strings.Contains(path, "/v4") {
						hasV4 = true
					}

					if firstFile == "" {
						pos := ctx.Fset.Position(imp.Pos())
						firstFile = pos.Filename
						firstLine = pos.Line
					}
				}
			}

			if !hasV3 || !hasV4 {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"V001", toolName,
				"Project mixes v3 and v4 go-cqrs-lite modules — "+
					"APIs are incompatible, migrate everything to v4",
				finding.SeverityError,
				finding.Pos(finding.FilePath(firstFile), firstLine, 1),
			).
				WithCategory(finding.CategoryCorrectness).
				WithConfidence(finding.ConfidenceHigh).
				WithFixStrategy(finding.FixStrategySuggest).
				WithSuggestion("Update all go-cqrs-lite imports to /v4 and run go mod tidy").
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
