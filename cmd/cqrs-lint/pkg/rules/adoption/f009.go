package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
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
			// Only relevant for server or command-dispatch projects — CLI
			// tools with simple timers don't need durable scheduling.
			if !ctx.FeatureProfile.HasServer &&
				ctx.FeatureProfile.CommandFlow != analyzer.CommandFlowCommands {
				return nil, nil
			}

			if importsPath(ctx, "go-cqrs-lite/scheduling") {
				return nil, nil
			}

			pos, ok := hasTimeBasedPatterns(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
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
			), nil
		},
	)
}
