package version

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// V006: Mixed version pins across go-cqrs-lite modules.
// Detects go-cqrs-lite modules within the same major version pinned to
// different minor/patch versions in the same go.mod. The modules are designed
// to work as a coherent set — mismatched versions cause subtle API
// incompatibilities (e.g., event/v4 at v4.2.0 but decider/v4 at v4.1.0).
//
//nolint:ireturn // factory returns public interface
func NewV006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V006-mixed-version-pins",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.ProjectRoot == "" {
				return nil, nil
			}

			requires := parseGoModCQRSRequires(ctx.ProjectRoot + "/go.mod")
			if len(requires) == 0 {
				return nil, nil
			}

			// Group non-pseudo versions by major version.
			versionByMajor := make(map[int]map[string][]cqrsRequire) // major → version → requires

			for _, req := range requires {
				if isPseudoVersion(req.Version) {
					continue
				}

				major, _, ok := majorMinorVersion(req.Version)
				if !ok {
					continue
				}

				if versionByMajor[major] == nil {
					versionByMajor[major] = make(map[string][]cqrsRequire)
				}

				versionByMajor[major][req.Version] = append(
					versionByMajor[major][req.Version], req,
				)
			}

			var findings []finding.Finding

			for major, versions := range versionByMajor {
				if len(versions) <= 1 {
					continue
				}

				sortedVersions := make([]string, 0, len(versions))
				for v := range versions {
					sortedVersions = append(sortedVersions, v)
				}
				slices.Sort(sortedVersions)

				// Report on the first (lowest) version line — that's what needs updating.
				lowestVersion := sortedVersions[0]
				flaggedReqs := versions[lowestVersion]
				if len(flaggedReqs) == 0 {
					continue
				}

				req := flaggedReqs[0]
				modName := shortModuleName(req.Path)

				f, err := finding.NewBuilder(
					"V006", toolName,
					fmt.Sprintf(
						"%s is on %s but other v%d modules use %s — "+
							"go-cqrs-lite modules within the same major version should be pinned to the same release",
						modName, req.Version, major,
						strings.Join(sortedVersions[1:], ", "),
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), req.Line, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(fmt.Sprintf(
						"Update all go-cqrs-lite/v%d modules to %s with: go get github.com/larsartmann/go-cqrs-lite/...@%s",
						major, sortedVersions[len(sortedVersions)-1], sortedVersions[len(sortedVersions)-1],
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
