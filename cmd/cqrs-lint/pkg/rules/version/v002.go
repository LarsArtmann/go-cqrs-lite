package version

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// V002: Unpinned go-cqrs-lite version.
// Detects go-cqrs-lite dependencies pinned to a pseudo-version (v0.0.0-*)
// instead of a tagged release. Pseudo-versions are non-reproducible across
// machines and indicate an unfinished or broken dependency setup.
//
//nolint:ireturn // factory returns public interface
func NewV002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V002-unpinned-version",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.ProjectRoot == "" {
				return nil, nil
			}

			requires := parseGoModCQRSRequires(ctx.ProjectRoot + "/go.mod")
			if len(requires) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, req := range requires {
				if !isPseudoVersion(req.Version) {
					continue
				}

				f, err := finding.NewBuilder(
					"V002", toolName,
					fmt.Sprintf(
						"%s is pinned to pseudo-version %s — use a tagged release for reproducibility",
						shortModuleName(req.Path), req.Version,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), req.Line, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(fmt.Sprintf(
						"Run go get %s@latest to pin a tagged release", req.Path)).
					WithSnippet(ctx.SourceLine(ctx.ProjectRoot+"/go.mod", req.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// isPseudoVersion returns true for Go pseudo-versions like
// "v0.0.0-00010101000000-000000000000" or "v0.0.0-20240115120000-abcdef123456".
func isPseudoVersion(v string) bool {
	return strings.HasPrefix(v, "v0.0.0-")
}
