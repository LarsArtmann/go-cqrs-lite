package architecture

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E001: Layer violation.
// Detects Tier 0 modules (id, codec, kv) importing Tier 3+ modules (decider, middleware).
func NewE001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E001-layer-violation",
		func(_ context.Context) ([]finding.Finding, error) {
			tier0Modules := map[string]bool{
				"go-cqrs-lite/id":         true,
				"go-cqrs-lite/codec":      true,
				"go-cqrs-lite/kv":         true,
				"go-cqrs-lite/dedup":      true,
				"go-cqrs-lite/dispatcher": true,
			}
			highTierPrefixes := []string{
				"decider", "middleware", "projectionhost", "scenario",
				"storage", "watermill", "transport", "catalog",
			}

			var findings []finding.Finding

			for _, pkg := range ctx.Packages {
				if pkg.PkgPath == "" || !isCQRSModule(pkg.PkgPath) {
					continue
				}

				pkgBase := strings.TrimPrefix(pkg.PkgPath, "github.com/larsartmann/go-cqrs-lite/")
				if !tier0Modules["go-cqrs-lite/"+pkgBase] {
					continue
				}

				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					impPath := strings.TrimPrefix(
						imp.PkgPath,
						"github.com/larsartmann/go-cqrs-lite/",
					)
					for _, high := range highTierPrefixes {
						if strings.HasPrefix(impPath, high) {
							f, err := finding.NewBuilder(
								"E001",
								toolName,
								fmt.Sprintf(
									"Tier-0 module %q imports Tier-3+ module %q — dependency direction violation",
									pkgBase,
									impPath,
								),
								finding.SeverityError,
								finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
							).
								WithCategory(finding.CategoryStructure).
								WithConfidence(finding.ConfidenceHigh).
								WithSuggestion("Tier 0 modules must have zero dependencies on higher tiers").
								Build()
							if err == nil {
								findings = append(findings, f)
							}

							break
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// E002: Circular dependency.
// Detects packages in the analyzed project that import each other.
func NewE002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E002-circular-dependency",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			seen := make(map[string]bool)

			for _, pkg := range ctx.Packages {
				if pkg.PkgPath == "" {
					continue
				}

				for _, imp := range pkg.Imports {
					if imp == nil || imp.PkgPath == "" {
						continue
					}

					for _, reversePkg := range ctx.Packages {
						if reversePkg.PkgPath != imp.PkgPath {
							continue
						}

						for _, reverseImp := range reversePkg.Imports {
							if reverseImp != nil && reverseImp.PkgPath == pkg.PkgPath {
								key := pkg.PkgPath + " <-> " + imp.PkgPath
								if seen[key] || seen[imp.PkgPath+" <-> "+pkg.PkgPath] {
									continue
								}

								seen[key] = true

								f, err := finding.NewBuilder(
									"E002", toolName,
									fmt.Sprintf("Circular dependency: %s ↔ %s", pkg.PkgPath, imp.PkgPath),
									finding.SeverityError,
									finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
								).
									WithCategory(finding.CategoryStructure).
									WithConfidence(finding.ConfidenceHigh).
									WithSuggestion("Break the cycle by extracting shared code into a third package").
									Build()
								if err == nil {
									findings = append(findings, f)
								}
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// E006: Event without projection.
// Detects event types emitted by deciders but not handled by any projection.
func NewE006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E006-event-without-projection",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			emittedTypes := make(map[string]analyzer.EventEmission)
			projectedTypes := make(map[string]bool)

			maps.Copy(emittedTypes, ctx.Registry.EventTypesEmitted)

			for _, proj := range ctx.Registry.Projections {
				for _, t := range proj.EventTypes {
					projectedTypes[t] = true
				}
			}

			for evtType, emission := range emittedTypes {
				if projectedTypes[evtType] {
					continue
				}

				pos := finding.Pos(finding.FilePath(emission.File), emission.Line, 1)
				if emission.File == "" {
					pos = finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1)
				}

				f, err := finding.NewBuilder(
					"E006", toolName,
					fmt.Sprintf("Event type %q is emitted but no projection handles it", evtType),
					finding.SeverityInfo,
					pos,
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceLow).
					WithSuggestion("Register a projection that handles this event type, or mark it as intentionally unhandled").
					WithSnippet(ctx.SourceLine(emission.File, emission.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

func isCQRSModule(path string) bool {
	prefix := "github.com/larsartmann/go-cqrs-lite"

	return path == prefix || (len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/")
}
