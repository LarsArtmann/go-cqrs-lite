package metaengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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

// catchUpMaxPasses bounds the stabilize loop: each pass replays everything
// that arrived during the previous one, so a write rate that persistently
// outpaces replay never converges. Past the bound the engine stays
// quarantined and the error tells the next StartAutoReprobe attempt to
// retry.
const catchUpMaxPasses = 64

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
//     The log keeps growing while the rebuild runs (the failover engine's
//     live writes still record into it), so replay drains suffixes in a
//     stabilize loop until a pass observes no growth.
//  3. Lift the quarantine ([Store.ReactivateEngine]) inside the same
//     append-blocked critical section that proves the log stopped growing:
//     with appends frozen, every event is either already replayed or is
//     recorded only after reactivation, where the live path folds it into
//     the now-active engine. No event can straddle the two.
//
// Any failure leaves the engine quarantined for the next attempt. The
// engine must currently be quarantined: while it is, folds reroute away
// from it, so the replay is the only writer to its collections until the
// gate (no double-folding). Requires an EventLog attached via
// [WithEventLog].
//
// [Store.StartAutoReprobe] runs this automatically when a quarantined
// engine's probe succeeds, falling back to a plain reactivation (with a
// warning) when catch-up is unsupported.
func (s *Store) CatchUpEngine(ctx context.Context, name string) error {
	replayed, err := s.catchUpEngine(ctx, name)

	// Hooks fire here, after the body released catchUpMu and every other
	// lock — observers must never run under Store locks.
	s.emitCatchUp(name, replayed, err)

	if err == nil {
		s.emitReactivated(name, "catchup")
	}

	return err
}

// catchUpEngine is the [Store.CatchUpEngine] body. The public wrapper emits
// the OnCatchUp/OnReactivated hooks after it returns, when no locks are held;
// the body itself holds catchUpMu until it returns.
func (s *Store) catchUpEngine(ctx context.Context, name string) (result error, replayed int) {
	s.catchUpMu.Lock()
	defer s.catchUpMu.Unlock()

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

	offset := 0

	// The rebuild is observable: mark it running, then settle the final
	// state on the way out (error text, replayed count, completion stamp).
	s.catchUpStateUpdate(name, func(rec *catchUpRecord) { rec.running = true })

	defer s.catchUpStateUpdate(name, func(rec *catchUpRecord) {
		rec.running = false

		switch {
		case result == nil:
			rec.lastErr = ""
			rec.replayed = replayed
			rec.completed = time.Now()
		case !errors.Is(result, ErrCatchUpUnsupported):
			rec.lastErr = result.Error()
		}
	})

	if err := resetter.ResetEngine(ctx); err != nil {
		return fmt.Errorf("metaengine.CatchUpEngine(%s): reset: %w", name, err)
	}

	for pass := 0; ; pass++ {
		events := s.eventLog.eventsFrom(offset)

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

		offset += len(events)
		replayed += len(events)

		// Stability gate: with appends blocked, "no growth across the last
		// pass" means the log held exactly the replayed prefix — and because
		// reactivation happens inside the same critical section, any event
		// recorded afterwards hits an active engine and folds live. Leaf
		// calls only inside the gate (ReactivateEngine takes healthMu; see
		// reactivateIfStable for the ordering contract).
		var reactivated bool

		grew := s.eventLog.reactivateIfStable(offset, func() {
			reactivated = s.reactivateEngine(name)
		})

		if !grew {
			if !reactivated {
				return fmt.Errorf(
					"metaengine.CatchUpEngine(%s): quarantine lifted during catch-up",
					name,
				)
			}

			break
		}

		if pass >= catchUpMaxPasses {
			return fmt.Errorf(
				"metaengine.CatchUpEngine(%s): event log kept growing across %d passes (writes outpace replay); engine stays quarantined for the next StartAutoReprobe attempt",
				name,
				pass+1,
			)
		}
	}

	slog.Info("metaengine: engine caught up and reactivated",
		"engine", name, "replayed_events", replayed)

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
		if s.reactivateEngine(name) {
			s.emitReactivated(name, "probe-fallback")
		}

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
