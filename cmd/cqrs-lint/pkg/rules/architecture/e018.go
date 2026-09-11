package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E018: Projection handles an event type that nothing emits.
// The mirror of E006 — E006 flags emitted-but-unhandled types, E018 flags
// handled-but-never-emitted ones: the typo class where a projection
// subscribes to "user.creted" and silently receives nothing forever. A type
// declared in the catalog counts as provided (external/imported events).
// With zero detected emissions the rule stays silent: the emission scanner
// only sees string-literal calls, so an empty producer set cannot
// distinguish a typo from an emission site it failed to parse.
//
// Fold cases are the C040 twin of this rule (same provider contract:
// emitted OR catalog-declared), keeping the three coeffect tiers in
// lockstep — runtime gate (system.DomainConfig.Events), static rules
// (E018 projections / C040 folds), docs side (catalog.ValidateCoeffects).
//
//nolint:ireturn // factory returns public interface
func NewE018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E018-projection-without-emitter",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			if len(ctx.Registry.EventTypesEmitted) == 0 {
				return findings, nil
			}

			for _, proj := range ctx.Registry.Projections {
				for _, evtType := range proj.EventTypes {
					if e018Provided(ctx, evtType) {
						continue
					}

					f, err := e018Finding(ctx, proj, evtType)
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// e018Provided reports whether any decider emits evtType or the catalog
// declares it (events imported from another service).
func e018Provided(ctx *analyzer.AnalysisContext, evtType string) bool {
	if _, ok := ctx.Registry.EventTypesEmitted[evtType]; ok {
		return true
	}

	return ctx.Registry.IsEventInCatalog(evtType)
}

// e018Finding builds the finding for one ghost subscription, anchored at the
// projection registration site.
func e018Finding(
	ctx *analyzer.AnalysisContext,
	proj analyzer.ProjectionInfo,
	evtType string,
) (finding.Finding, error) {
	pos := finding.Pos(finding.FilePath(proj.File), proj.Pos.Line, proj.Pos.Column)
	if proj.File == "" {
		pos = finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1)
	}

	name := proj.Name
	if name == "" {
		name = "subscription"
	}

	f, err := finding.NewBuilder(
		"E018", toolName,
		fmt.Sprintf(
			"Projection %q handles event type %q that no decider emits and the catalog does not declare — possible typo",
			name,
			evtType,
		),
		finding.SeverityWarning,
		pos,
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion(
			"Fix the typo, emit it via event.New, or catalog.Event it when imported from another service; " +
				"system.New's coeffect gate (DomainConfig.Events) enforces the same contract at runtime",
		).
		WithSnippet(ctx.SourceLine(proj.File, proj.Pos.Line)).
		Build()

	return f, err
}
