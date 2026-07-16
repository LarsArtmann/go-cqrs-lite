package correctness

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects fold functions whose switch default case returns nil error.
func NewC003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C003-silent-unknown-event-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if !fold.HasSwitch || !fold.HasDefault || !fold.DefaultNil {
					continue
				}

				f, err := finding.NewBuilder(
					"C003", toolName,
					fmt.Sprintf("Fold %s silently ignores unknown event types in default case", fold.FuncName),
					finding.SeverityError,
					finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategyDirect).
					WithSuggestion("Return an error in the default case: return state, fmt.Errorf(\"fold: unknown event type: %s\", evt.Type())").
					WithBeforeCode("return state, nil").
					WithAfterCode(`return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())`).
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
