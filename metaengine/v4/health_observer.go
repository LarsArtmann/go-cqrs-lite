package metaengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// Health-transition hook emission (ADR-0137 observability). Each helper is
// nil-checked so unconfigured hooks cost one pointer comparison — the same
// zero-overhead contract as the fold/execute hooks.
//
// Re-entrancy contract: helpers are invoked after the health mutex is
// released. OnQuarantined may still fire while a caller holds the Store's
// read lock (execute paths record failures under s.mu R) — observers must
// not call back into the Store. Meters and loggers are always safe.

func (s *Store) emitQuarantined(engine string, failures int, lastErr string) {
	if h := s.hooks; h != nil && h.OnQuarantined != nil {
		h.OnQuarantined(engine, failures, lastErr)
	}
}

func (s *Store) emitReactivated(engine, reason string) {
	if h := s.hooks; h != nil && h.OnReactivated != nil {
		h.OnReactivated(engine, reason)
	}
}

func (s *Store) emitProbe(engine string, err error) {
	if h := s.hooks; h != nil && h.OnProbe != nil {
		h.OnProbe(engine, err)
	}
}

func (s *Store) emitCatchUp(engine string, replayed int, err error) {
	if h := s.hooks; h != nil && h.OnCatchUp != nil {
		h.OnCatchUp(engine, replayed, err)
	}
}

// catchUpReplayedCount reads how many events the last CatchUpEngine attempt
// replayed into the engine (partial on failure) — the OnCatchUp payload.
func (s *Store) catchUpReplayedCount(name string) int {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()

	if rec := s.catchUps[name]; rec != nil {
		return rec.replayed
	}

	return 0
}

// doctorEngineHealthSection renders the ADR-0137 per-engine health lines for
// the Doctor report: quarantine state, consecutive failures, and the last
// classified error, sorted by engine name.
func (s *Store) doctorEngineHealthSection() string {
	health := s.HealthSnapshot()
	if len(health) == 0 {
		return "  no failures recorded\n"
	}

	names := slices.Sorted(maps.Keys(health))

	var b strings.Builder

	for _, name := range names {
		h := health[name]

		switch {
		case h.State == EngineQuarantined:
			fmt.Fprintf(&b, "  %s: QUARANTINED (failures=%d, since=%s, last=%q)\n",
				name, h.ConsecutiveFailures, h.QuarantinedAt.Format(time.RFC3339), h.LastError)

		case h.ConsecutiveFailures > 0:
			fmt.Fprintf(
				&b,
				"  %s: active (failures=%d, last=%q)\n",
				name,
				h.ConsecutiveFailures,
				h.LastError,
			)

		default:
			fmt.Fprintf(&b, "  %s: active\n", name)
		}
	}

	return b.String()
}
