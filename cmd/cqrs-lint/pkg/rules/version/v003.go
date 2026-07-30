package version

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// latestKnownMinor is the latest known minor version of the v4 series.
// Update this constant when a new minor release ships.
// As of 2026-07, the latest published modules are v4.3.x.
const latestKnownMinor = 3

// V003: Version lag behind latest.
// Detects go-cqrs-lite v4 dependencies more than 2 minor versions behind the
// latest known release. Stale versions miss bug fixes, security patches, and
// new features. Only direct (non-indirect) requires are checked — indirect
// deps are resolved by Go's MVS to the maximum compatible version.
//
//nolint:ireturn // factory returns public interface
func NewV003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V003-version-lag",
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
				if req.Indirect {
					continue
				}

				if isPseudoVersion(req.Version) {
					continue
				}

				major, minor, ok := majorMinorVersion(req.Version)
				if !ok || major != 4 {
					continue
				}

				lag := latestKnownMinor - minor
				if lag <= 2 {
					continue
				}

				f, err := finding.NewBuilder(
					"V003", toolName,
					fmt.Sprintf(
						"%s is on v4.%d.x — %d minor versions behind latest (v4.%d.x), missing bug fixes and features",
						shortModuleName(req.Path), minor, lag, latestKnownMinor,
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), req.Line, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(fmt.Sprintf(
						"Run go get %s@latest to update to the latest release", req.Path)).
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
