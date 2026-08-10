package adoption

import (
	"context"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F001 detects projects that have Delete* functions and emit events but do not
// use domain deletion events (e.g., "user.deleted"). Per ADR-0114, tombstones
// are domain events — emitting a deletion event preserves the audit trail and
// enables replay-safe deletion semantics.
//
//nolint:ireturn // factory returns public interface
func NewF001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F001-no-domain-delete-event",
		func(_ context.Context) ([]finding.Finding, error) {
			if eventCount(ctx) == 0 {
				return nil, nil
			}

			if hasDeletionEventTypes(ctx) {
				return nil, nil
			}

			pos, ok := firstFuncDeclPos(ctx, "Delete")
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F001",
				"Delete operations detected but no domain deletion event found — "+
					"consider emitting a deletion event (e.g., \"user.deleted\") for audit trail and replay safety",
				"Per ADR-0114, deletion is expressed as a domain event in the "+
					"same stream. Emit a \"<aggregate>.deleted\" event type instead "+
					"of hard deletes. This preserves the event audit trail.",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}

// hasDeletionEventTypes reports whether the project emits any event type
// containing "delete" or "deleted" (case-insensitive).
func hasDeletionEventTypes(ctx *analyzer.AnalysisContext) bool {
	for eventType := range ctx.Registry.EventTypesEmitted {
		lower := strings.ToLower(eventType)
		if strings.Contains(lower, "delete") || strings.Contains(lower, "deleted") {
			return true
		}
	}

	return false
}
