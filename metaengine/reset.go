package metaengine

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// EngineResetter is an optional engine capability: engines that can drop ALL
// their materialized state and return to the empty, post-construction state
// implement it. [Store.Reset] uses it to fully revert a projection so a replay
// from the beginning of the journal rebuilds the read model from zero.
//
// Engines that do NOT implement EngineResetter cannot be bulk-cleared;
// [Store.Reset] reports them in [ResetResult.UnclearableEngines] so a partial
// reset is never silent. Every first-party engine implements it — memory,
// SQLite/Turso, Pebble, bbolt, Badger, Postgres, MySQL, DuckDB, Dgraph, and
// the iroh replication wrapper (via its local engine); custom engines opt in
// by implementing this interface.
type EngineResetter interface {
	ResetEngine(ctx context.Context) error
}

// ResetResult reports the outcome of a [Store.Reset]. A reset is PARTIAL when at
// least one engine could not be bulk-cleared (it lacks [EngineResetter]); those
// engines still hold stale materialized state that a replay will be applied on
// top of. ClearedEngines lists the engines that were emptied.
type ResetResult struct {
	ClearedEngines     []string
	UnclearableEngines []string
}

// Partial reports whether any engine could not be bulk-cleared, meaning stale
// read-model state may remain after the reset.
func (r ResetResult) Partial() bool { return len(r.UnclearableEngines) > 0 }

// String renders a short human summary suitable for logs and error messages.
func (r ResetResult) String() string {
	if !r.Partial() {
		return fmt.Sprintf("cleared %d engine(s): %s",
			len(r.ClearedEngines), strings.Join(r.ClearedEngines, ", "))
	}

	return fmt.Sprintf("cleared [%s]; could NOT bulk-clear [%s] — stale read-model state remains",
		strings.Join(r.ClearedEngines, ", "), strings.Join(r.UnclearableEngines, ", "))
}

// Reset reverts the Store to its empty, post-Plan state so a replay from the
// beginning of the journal rebuilds every materialized collection from zero. It
// clears the recorded event log, the idempotency dedup window, and the poison
// tracker (replay-affecting state that must never survive a reset), then asks
// each engine to drop its materialized data via the optional [EngineResetter]
// capability.
//
// The returned [ResetResult] describes which engines were cleared and which
// could not be (see [ResetResult.Partial]). A non-nil error is returned only
// when an engine's ResetEngine fails; the internal replay-affecting state is
// cleared even if some engine is unclearable, so callers should inspect both.
//
// This is the metaengine primitive behind projectionhost's Resettable contract:
// projectionadapter.Adapter.Reset delegates here so a Host.Reset performs a
// one-call revert of a metaengine-backed projection.
func (s *Store) Reset(ctx context.Context) (ResetResult, error) {
	s.mu.RLock()
	engines := slices.Clone(s.engines)
	s.mu.RUnlock()

	// A reset means "rebuild from zero", so replay-affecting internal state must
	// not survive regardless of engine capability.
	if s.eventLog != nil {
		s.eventLog.Clear()
	}

	if s.idempotency != nil {
		s.idempotency.Clear()
	}

	if s.poison != nil {
		s.poison.Clear()
	}

	var result ResetResult

	for _, eng := range engines {
		name := eng.Profile().Name
		if name == "" {
			name = "engine"
		}

		resetter, ok := eng.(EngineResetter)
		if !ok {
			result.UnclearableEngines = append(result.UnclearableEngines, name)

			continue
		}

		if err := resetter.ResetEngine(ctx); err != nil {
			return result, fmt.Errorf("metaengine: reset engine %q: %w", name, err)
		}

		result.ClearedEngines = append(result.ClearedEngines, name)
	}

	return result, nil
}
