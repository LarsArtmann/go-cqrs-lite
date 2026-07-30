package boilerplate

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B026: Manual event type registration instead of catalog.
// Detects projects with 3+ event type string constants but no catalog import.
// Without catalog registration, event documentation and OpenAPI/AsyncAPI
// generation is unavailable.
//
//nolint:ireturn // factory returns public interface
func NewB026Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B026-missing-catalog-registration",
		func(_ context.Context) ([]finding.Finding, error) {
			// Use the registry's EventTypesEmitted (event types from event.New calls)
			// as a proxy for event type string constants.
			emittedCount := len(ctx.Registry.EventTypesEmitted)
			if emittedCount < 3 {
				return nil, nil
			}

			// Check if any package imports the catalog module.
			hasCatalogImport := false
			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "go-cqrs-lite/catalog") {
						hasCatalogImport = true
						break
					}
				}

				if hasCatalogImport {
					break
				}
			}

			if hasCatalogImport {
				return nil, nil
			}

			var findings []finding.Finding

			// Report on the go.mod file — this is a project-level finding.
			f, err := finding.NewBuilder(
				"B026", toolName,
				"Project has 3+ event types but no catalog import — "+
					"event documentation and OpenAPI/AsyncAPI generation unavailable",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithFixStrategy(finding.FixStrategySuggest).
				WithSuggestion("Import catalog and register event types with catalog.NewBuilder() — " +
					"enables auto-generated AsyncAPI, OpenAPI, D2, and EventCatalog documentation").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
