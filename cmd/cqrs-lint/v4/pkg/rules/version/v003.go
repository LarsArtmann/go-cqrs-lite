package version

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// V003: Version lag behind latest.
// Detects go-cqrs-lite v4 dependencies more than 2 minor versions behind
// the module's OWN latest release (resolved per module path from the Go
// module proxy). A verdict requires VERIFIED proxy truth: when the proxy
// is unreachable, disabled, or bypassed (GOPRIVATE/GONOPROXY), the rule
// stays silent — a fleet-constant fallback fabricated "behind latest"
// claims that were false positives by construction (issue #129; the
// systemcatalog/v4 v4.0.0 pin mislabeled "3 behind v4.3.x" in every
// GOPRIVATE-covered fleet run).
// Stale versions miss bug fixes, security patches, and new features. Only
// direct (non-indirect) requires are checked — indirect deps are resolved
// by Go's MVS to the maximum compatible version.
//
//nolint:ireturn // factory returns public interface
func NewV003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return newV003Detector(ctx, latestMinorFetch)
}

// NewV003DetectorWithLatest builds the V003 detector with an injected
// per-module latest lookup — the seam that exercises the proxy path
// deterministically (the default fetch is a no-op inside test binaries).
//
//nolint:ireturn // factory returns public interface
func NewV003DetectorWithLatest(
	ctx *analyzer.AnalysisContext,
	fetch func(context.Context, string) (int, bool),
) finding.Detector {
	return newV003Detector(ctx, fetch)
}

//nolint:ireturn // factory returns public interface
func newV003Detector(
	ctx *analyzer.AnalysisContext,
	fetch func(context.Context, string) (int, bool),
) finding.Detector {
	return finding.NamedDetectorFunc(
		"V003-version-lag",
		func(detectorCtx context.Context) ([]finding.Finding, error) {
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

				latestMinor, verified := latestMinorFor(detectorCtx, req.Path, fetch)
				if !verified {
					continue
				}

				lag := latestMinor - minor
				if lag <= 2 {
					continue
				}

				f, err := findingTemplate.Builder(
					"V003",
					fmt.Sprintf(
						"%s is on v4.%d.x — %d minor versions behind latest (%s), missing bug fixes and features",
						shortModuleName(req.Path),
						minor,
						lag,
						renderLatestMinor(latestMinor),
					),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), req.Line, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(fmt.Sprintf(
						"Run go get %s@latest to update to the latest release", req.Path,
					)).
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
