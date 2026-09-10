package projectionadapter

import (
	"context"
	"fmt"
)

// Reset implements projectionhost.Resettable so a Host.Reset performs a
// one-call revert of a metaengine-backed projection: it delegates to
// [metaengine.Store.Reset], which clears the store's replay-affecting state
// (event log, idempotency window, poison marks) and asks each engine to drop
// its materialized collections. After Reset, the next Start replays the journal
// from zero and rebuilds the read model cleanly.
//
// Engines that implement metaengine.EngineResetter (the in-memory engine today)
// are fully cleared. Engines that do not (persistent engines such as SQLite are
// documented follow-ups) cannot be bulk-cleared; Reset logs a warning naming
// them and still returns nil, because the checkpoint-only reset remains useful
// and a hard failure would break existing v4 callers. Inspect the warning — or
// call metaengine.Store.Reset directly for the structured ResetResult — when a
// full revert matters. In v5 a partial reset becomes a hard error (it rides the
// ADR-0123 composition-root wave).
func (a *Adapter) Reset(ctx context.Context) error {
	result, err := a.store.Reset(ctx)
	if err != nil {
		return fmt.Errorf("projectionadapter: reset store %q: %w", a.name, err)
	}

	if result.Partial() {
		a.logger.Warn(
			"projectionadapter: Reset cleared the checkpoint and store state but some engines could not be bulk-cleared, so stale read-model state may remain for the replay",
			"projection",
			a.name,
			"result",
			result.String(),
			"remedy",
			"use an engine that implements metaengine.EngineResetter (the in-memory engine does), or clear the persistent engine's tables out-of-band before replaying",
		)
	}

	return nil
}
