package boilerplate

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects decider fold/apply functions that don't use decider.StrictApply.
// Without StrictApply, unknown event types are silently ignored, hiding
// bugs when new event types are added.
//
// B021: Missing StrictApply in fold functions.
//
//nolint:ireturn // factory returns public interface
func NewB021Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B021-fold-without-strictapply",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.DefaultNil {
					continue
				}

				if ctx.Registry.StrictApplyFolds[fold.FuncName] {
					continue
				}

				f, err := finding.NewBuilder(
					"B021", toolName,
					"Fold function silently ignores unknown event types — "+
						"use decider.StrictApply for compile-time safety",
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Wrap the fold in decider.StrictApply(fold, []event.Type{...}) "+
						"so adding a new event type without handling it becomes a compile-time error.").
					WithSnippet(ctx.SourceLine(fold.File, fold.Pos.Line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
