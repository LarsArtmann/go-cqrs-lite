package metaengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ErrCatchUpUnsupported reports that an engine cannot be automatically
// rebuilt: it lacks [EngineResetter], or the Store has no EventLog attached
// (the journal is the rebuild source). Callers may fall back to a plain
// [Store.ReactivateEngine], accepting that the engine serves stale
// read-model state until it is rebuilt out-of-band.
var ErrCatchUpUnsupported = errors.New(
	"metaengine: engine catch-up unsupported (no EngineResetter or no EventLog)",
)

// CatchUpEngine rebuilds a QUARANTINED engine's materialized collections and
// only then lifts the quarantine — the write-failover recovery half of
// ADR-0137. While an engine is quarantined its folds are rerouted to a
// healthy failover engine, so the quarantined engine's own state goes stale;
// reactivating it without a rebuild would serve those stale collections the
// moment it is trusted again.
//
// The rebuild is the ADR-0136 reset+replay primitive applied to one engine:
//
//  1. Reset the engine via [EngineResetter] (every first-party engine
//     implements it) — from empty, non-idempotent folds replay cleanly.
//  2. Replay the attached EventLog into EXACTLY this engine's queries
//     (healthy engines never see the replay, so nothing double-applies).
//  3. Lift the quarantine ([Store.ReactivateEngine]) only after a clean
//     rebuild; any failure leaves the engine quarantined for the next
//     attempt.
//
// The engine must currently be quarantined: while it is, reads and folds
// reroute away from it, so the replay is the only writer and no live apply
// can interleave with the rebuild (no double-folding). Requires an EventLog
// attached via [WithEventLog].
//
// [Store.StartAutoReprobe] runs this automatically when a quarantined
// engine's probe succeeds, falling back to a plain reactivation (with a
// warning) when catch-up is unsupported.
func (s *Store) CatchUpEngine(ctx context.Context, name string) error {
	s.mu.RLock()
	var target Engine

	for _, eng := range s.engines {
		if eng.Profile().Name == name {
			target = eng

			break
		}
	}
	s.mu.RUnlock()

	if target == nil {
		return fmt.Errorf("metaengine.CatchUpEngine: unknown engine %q", name)
	}

	if !s.engineQuarantined(name) {
		return fmt.Errorf(
			"metaengine.CatchUpEngine(%s): engine is not quarantined — catch-up rebuilds an engine whose folds were rerouted away while quarantined",
			name,
		)
	}

	resetter, ok := target.(EngineResetter)
	if !ok {
		return fmt.Errorf("%w: %T does not implement EngineResetter", ErrCatchUpUnsupported, target)
	}

	if s.eventLog == nil {
		return fmt.Errorf("%w: attach one via WithEventLog", ErrCatchUpUnsupported)
	}

	events := s.eventLog.Events()

	if err := resetter.ResetEngine(ctx); err != nil {
		return fmt.Errorf("metaengine.CatchUpEngine(%s): reset: %w", name, err)
	}

	for _, evt := range events {
		rec := evt.Record
		if rec.Type == "" {
			rec = record.Record{Type: evt.Type}
		}

		if err := s.dispatchFoldsToEngine(ctx, target, evt.Type, rec, evt.Payload); err != nil {
			return fmt.Errorf(
				"metaengine.CatchUpEngine(%s): replay event %q: %w (engine stays quarantined; the next StartAutoReprobe attempt retries)",
				name,
				evt.Type,
				err,
			)
		}
	}

	if !s.ReactivateEngine(name) {
		return fmt.Errorf("metaengine.CatchUpEngine(%s): quarantine lifted during catch-up", name)
	}

	slog.Info("metaengine: engine caught up and reactivated",
		"engine", name, "replayed_events", len(events))

	return nil
}

// catchUpOrReactivate is the StartAutoReprobe completion: prefer a consistent
// rebuild, fall back to a plain reactivation (preserving the pre-failover
// behavior) when catch-up is structurally impossible, and stay quarantined
// when a rebuild attempt failed midway.
func (s *Store) catchUpOrReactivate(ctx context.Context, name string) {
	err := s.CatchUpEngine(ctx, name)

	switch {
	case err == nil:
		return
	case errors.Is(err, ErrCatchUpUnsupported):
		s.ReactivateEngine(name)

		slog.Warn(
			"metaengine: engine answered its probe but catch-up is unsupported; reactivated without rebuild — its read models may be stale until rebuilt out-of-band",
			"engine",
			name,
			"cause",
			err,
		)
	default:
		slog.Error("metaengine: engine answered its probe but catch-up failed; staying quarantined",
			"engine", name, "error", err)
	}
}
