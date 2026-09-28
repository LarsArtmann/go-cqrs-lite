package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F009 detects projects with time-based business rules (deadlines, timeouts,
// expirations) that do not use the scheduling module. Rolling your own timers
// with time.AfterFunc is fragile — the scheduling module provides durable,
// idempotent deadline management.
//
//nolint:ireturn // factory returns public interface
func NewF009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F009-no-scheduling-module",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if !sc.profile.HasServer &&
					sc.profile.CommandFlow != analyzer.CommandFlowCommands {
					continue
				}

				if importsPathIn(sc.files, "go-cqrs-lite/scheduling") {
					continue
				}

				pos, ok := hasTimeBasedPatternsIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F009",
					"Time-based patterns detected (timers, deadlines, expirations) "+
						"but scheduling module is not used — hand-rolled timers are "+
						"fragile and not crash-safe",
					"Import the scheduling module and use scheduling.Scheduler with "+
						"a TimerStore for durable, idempotent deadline management. "+
						"Example: scheduling.New(store, dispatchFn) for 'cancel order "+
						"after 30 min' patterns.",
					pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
		},
	)
}
