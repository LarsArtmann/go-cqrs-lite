package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E021: an event type the project emits carries no schema declaration.
// The declaration list (schema.Event/EventOf, or the system.Schemas()
// builder feeding DomainConfig.Schema) is what drives upcasting and typed
// projection decoding — an emitted type outside it is the typo class that
// silently skips the whole evolution machinery ("user.creted" emitted while
// "user.created" is declared decodes as raw bytes forever). Scopes to LOCALLY
// EMITTED types: consumption-only types are typically imported from another
// service, where E018's provider contract (emitted or catalog-declared) is
// the governing tier — flagging them here would double-report E018's typo
// and punish legitimate external events. catalog-declared types count as
// declared (the same provided-contract as E018 — the operator surfaced the
// type in governance). With zero detected declarations the rule stays
// silent: a project that has not adopted schema declarations must not be
// coached into them (E018's zero-silence convention).
//
// The third lockstep tier beside the runtime coeffect gate
// (system.DomainConfig.Events) and E018/C040 provider parity.
//
//nolint:ireturn // factory returns public interface
func NewE021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E021-emitted-without-schema-declaration",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			if len(ctx.Registry.EventTypesInSchemaDecl) == 0 {
				return findings, nil
			}

			for evtType, emission := range ctx.Registry.EventTypesEmitted {
				if ctx.Registry.IsEventSchemaDeclared(evtType) ||
					ctx.Registry.IsEventInCatalog(evtType) {
					continue
				}

				f, err := e021Finding(ctx, evtType, emission)
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// e021Finding builds the finding for one undeclared emitted type, anchored
// at its emission site.
func e021Finding(
	ctx *analyzer.AnalysisContext,
	evtType string,
	emission analyzer.EventEmission,
) (finding.Finding, error) {
	file := emission.File
	if file == "" {
		file = ctx.ProjectRoot + "/go.mod"
	}

	return findingTemplate.Builder(
		"E021",
		fmt.Sprintf(
			"event type %q is emitted but carries no schema declaration — upcasting and typed decoding skip it (typo class: a near-identical declared type never matches)",
			evtType,
		),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(file), emission.Line, 1),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion(
			"Declare it where the other schemas are declared (schema.EventOf, or the system.Schemas() builder's .Event) — "+
				"or fix the typo if a near-identical type is already declared; "+
				"catalog.Event counts as declared when the type is intentionally external",
		).
		WithSnippet(ctx.SourceLine(emission.File, emission.Line)).
		Build()
}
