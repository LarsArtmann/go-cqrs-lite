package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F001 detects projects that have Delete* functions and emit events but do not
// use event.MarkTombstone for soft-delete. Tombstone metadata preserves the
// audit trail and enables replay-safe deletion semantics.
//
//nolint:ireturn // factory returns public interface
func NewF001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F001-no-tombstone-softdelete",
		func(_ context.Context) ([]finding.Finding, error) {
			if eventCount(ctx) == 0 {
				return nil, nil
			}

			if projectHasCall(ctx, "event", "MarkTombstone") ||
				projectHasCall(ctx, "event", "DetectTombstone") {
				return nil, nil
			}

			pos, ok := firstFuncDeclPos(ctx, "Delete")
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F001",
				"Delete operations detected but event.MarkTombstone is not used — "+
					"consider tombstone soft-delete for audit trail and replay safety",
				"Use event.MarkTombstone(evt) to soft-delete via tombstone metadata "+
					"instead of hard deletes. This preserves the event audit trail.",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}
