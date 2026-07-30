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
			var firstPos analyzer.GoFile
			var firstPkgPath string

			for _, gf := range ctx.GoFiles {
				if gf.Pkg == nil {
					continue
				}

				for _, imp := range gf.Pkg.Imports {
					if imp == nil || !analyzer.IsCQRSModulePath(imp.PkgPath) {
						continue
					}

					if strings.Contains(imp.PkgPath, "/v3") {
						hasV3 = true
					}

					if strings.Contains(imp.PkgPath, "/v4") {
						hasV4 = true
					}

					if firstPkgPath == "" {
						firstPkgPath = imp.PkgPath
						firstPos = *gf
					}
				}
			}

			if !hasV3 || !hasV4 {
				return nil, nil
			}

			var findings []finding.Finding

			pos := ctx.Fset.Position(firstPos.AST.Pos())
			f, err := finding.NewBuilder(
				"V001", toolName,
				"Project mixes v3 and v4 go-cqrs-lite modules — "+
					"APIs are incompatible, migrate everything to v4",
				finding.SeverityError,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryCorrectness).
				WithConfidence(finding.ConfidenceHigh).
				WithFixStrategy(finding.FixStrategySuggest).
				WithSuggestion("Update all go-cqrs-lite imports to /v4 and run go mod tidy").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
