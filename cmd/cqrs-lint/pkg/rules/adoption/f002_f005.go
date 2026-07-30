package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F002 detects projects with 3+ event types that do not call catalog.NewBuilder.
// The catalog module generates AsyncAPI, OpenAPI, D2, and EventCatalog docs.
//
//nolint:ireturn // factory returns public interface
func NewF002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F002-no-catalog-documentation",
		func(_ context.Context) ([]finding.Finding, error) {
			if eventCount(ctx) < 3 {
				return nil, nil
			}

			if projectHasCall(ctx, "catalog", "NewBuilder") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F002",
				"Project has "+itoa(eventCount(ctx))+
					" event types but no catalog.NewBuilder — event documentation "+
					"is not generated",
				"Import catalog and call catalog.NewBuilder() to generate AsyncAPI, "+
					"OpenAPI, D2 diagrams, and EventCatalog documentation from your "+
					"event types.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F005 detects projects with evolving event schemas (WithSchemaVersion calls
// or 5+ event types) that do not use schema.NewUpcaster for version migration.
//
//nolint:ireturn // factory returns public interface
func NewF005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F005-no-schema-upcasters",
		func(_ context.Context) ([]finding.Finding, error) {
			if projectHasCall(ctx, "schema", "NewUpcaster") {
				return nil, nil
			}

			// Signal 1: explicit WithSchemaVersion usage (schemas are versioned).
			hasVersioning := projectHasCall(ctx, "event", "WithSchemaVersion")

			// Signal 2: many event types suggest a complex, evolving domain.
			if eventCount(ctx) < 5 && !hasVersioning {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			msg := "Project has evolving event schemas but no schema.NewUpcaster — " +
				"schema migrations on load are not handled"

			if hasVersioning {
				msg = "Project uses event.WithSchemaVersion but has no schema.NewUpcaster — " +
					"versioned events need upcasters for backward-compatible migration"
			}

			return singleInfoFinding(ctx, "F005", msg,
				"Use schema.NewUpcaster(eventType, fromVersion, migrateFn) to transform "+
					"old event payloads to the current schema on load. Register upcasters "+
					"via schema.NewVersionedStore.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// itoa is a thin wrapper to avoid importing strconv in every rule file.
func itoa(n int) string { return strconvItoa(n) }
