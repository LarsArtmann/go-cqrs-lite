package correctness

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// C040: Dead fold case detection (reverse direction of C038).
//
// C038 checks "emitted type not handled by any fold" (emit → fold direction).
// C040 checks "fold handles a type that nobody ever emits" (fold → emit
// direction). A dead fold case is either leftover from a removed/renamed
// event, or the fold case string itself has a typo.
//
// Provider parity with E018 and the runtime coeffect gate: a type declared
// via catalog.Event counts as provided (events imported from another
// service), so an imported fold case is NOT dead code.
//
// Safety: C040 only defers to C038 when a near-miss emitter exists AND that
// near-miss type is not itself handled by a fold case. When the near-miss IS
// handled (the fold carries both "user.created" and its typo "user.creted"),
// C038 sees the emission as handled and stays silent — C040 fires there
// instead, so a typo'd case cannot hide behind its corrected twin. C040 also
// suppresses entirely when no emissions are detected (cross-module safety:
// the linter may only see the fold side).
//
//nolint:ireturn // factory returns public interface
func NewC040Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C040-dead-fold-case",
		func(_ context.Context) ([]finding.Finding, error) {
			emitted := ctx.Registry.EventTypesEmitted
			if len(emitted) == 0 {
				return nil, nil
			}

			foldCases := ctx.CollectFoldCasesWithPos()
			if len(foldCases) == 0 {
				return nil, nil
			}

			emittedList := make([]string, 0, len(emitted))
			for t := range emitted {
				emittedList = append(emittedList, t)
			}

			emittedSet := make(map[string]bool, len(emitted))
			for t := range emitted {
				emittedSet[t] = true
			}

			handledSet := make(map[string]bool, len(foldCases))
			for _, fc := range foldCases {
				handledSet[fc.Value] = true
			}

			var findings []finding.Finding

			for _, fc := range foldCases {
				if emittedSet[fc.Value] || ctx.Registry.IsEventInCatalog(fc.Value) {
					continue
				}

				if closest, dist := nearestMatch(fc.Value, emittedList); dist <= 2 && !handledSet[closest] {
					// A near-miss emitter exists and nothing folds it — C038
					// reports the mismatch from the emit side.
					continue
				}

				f, err := finding.NewBuilder(
					"C040", toolName,
					fmt.Sprintf(
						"Fold case %q in %s is never emitted via event.New and the catalog does not declare it — "+
							"dead code or a typo in the fold case string",
						fc.Value, fc.FoldName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(fc.File), fc.Pos.Line, fc.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(fmt.Sprintf(
						"Remove the case for %q, fix the typo, or catalog.Event it when imported from another service; "+
							"system.New's coeffect gate (DomainConfig.Events) enforces the same contract at runtime",
						fc.Value,
					)).
					WithSnippet(ctx.SourceLine(fc.File, fc.Pos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// C040's fold cases come from the shared analyzer collector
// ([analyzer.AnalysisContext.CollectFoldCasesWithPos]), which also powers
// C038 and resolves const-identifier case labels to their string values.
