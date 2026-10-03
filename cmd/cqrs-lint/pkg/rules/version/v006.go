package version

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// V006: Divergent pins of the same go-cqrs-lite module across a workspace.
//
// go-cqrs-lite tags are PER-MODULE (event/v4, decider/v4, system/v4, … each
// carry their own release numbers; there is no unified release), so different
// minor/patch versions ACROSS modules in one go.mod are normal and correct.
// What IS a defect: the SAME module path pinned at different versions in
// different go.mod files of one repository. Workspace (go.work) builds hide
// the divergence behind MVS, but GOWORK=off builds — CI module matrices,
// nix FODs, per-module tooling — resolve each go.mod independently and fail
// with "updates to go.mod needed" until the pins are realigned in the same
// change (the workspace-lockstep class).
//
//nolint:ireturn // factory returns public interface
func NewV006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"V006-divergent-module-pins",
		func(_ context.Context) ([]finding.Finding, error) {
			if ctx.ProjectRoot == "" {
				return nil, nil
			}

			goModFiles, err := findGoModFiles(ctx.ProjectRoot)
			if err != nil || len(goModFiles) == 0 {
				return nil, nil
			}

			// pinSites maps module path → pins, each pin remembering where it
			// was declared so the finding can anchor on the stale line.
			type pinSite struct {
				version string
				file    string // absolute path of the declaring go.mod
				line    int
			}

			pinSites := make(map[string][]pinSite)

			for _, goMod := range goModFiles {
				for _, req := range parseGoModCQRSRequires(goMod) {
					if isPseudoVersion(req.Version) {
						continue
					}

					pinSites[req.Path] = append(pinSites[req.Path], pinSite{
						version: req.Version,
						file:    goMod,
						line:    req.Line,
					})
				}
			}

			var findings []finding.Finding

			for _, path := range sortedKeys(pinSites) {
				sites := pinSites[path]
				if len(sites) < 2 {
					continue
				}

				// Sort pins by version ascending; the finding anchors on the
				// lowest (stale) pin and suggests the highest.
				sorted := make([]pinSite, len(sites))
				copy(sorted, sites)
				slices.SortStableFunc(sorted, func(a, b pinSite) int {
					return semverCompare(a.version, b.version)
				})

				if sorted[0].version == sorted[len(sorted)-1].version {
					continue
				}

				lowest := sorted[0]
				highest := sorted[len(sorted)-1]

				others := make([]string, 0, len(sorted)-1)
				for _, s := range sorted[1:] {
					others = append(others, fmt.Sprintf("%s in %s",
						s.version, relDisplay(ctx.ProjectRoot, s.file)))
				}

				f, err := findingTemplate.Builder(
					"V006",
					fmt.Sprintf(
						"%s is pinned at %s in %s but %s — the same go-cqrs-lite "+
							"module must be pinned consistently across the workspace "+
							"(GOWORK=off builds resolve each go.mod independently)",
						shortModuleName(path), lowest.version,
						relDisplay(ctx.ProjectRoot, lowest.file),
						strings.Join(others, ", "),
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(lowest.file), lowest.line, 1),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion(fmt.Sprintf(
						"Align the pin in %s: go get %s@%s (bump every affected "+
							"go.mod in the same change)",
						relDisplay(ctx.ProjectRoot, lowest.file), path, highest.version,
					)).
					WithSnippet(ctx.SourceLine(lowest.file, lowest.line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// findGoModFiles walks the project root and returns every go.mod path,
// skipping vendor/.git/node_modules/testdata/dist/build directories (same
// skip set as the analyzer's module discovery).
func findGoModFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip inaccessible paths, continue walking
		}

		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "testdata", "dist", "build":
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() == "go.mod" {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	return files, nil
}

// relDisplay renders a go.mod path relative to the project root when
// possible (shorter findings); falls back to the absolute path.
func relDisplay(root, file string) string {
	if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}

	return file
}

// sortedKeys returns the map's keys sorted for deterministic finding order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}
