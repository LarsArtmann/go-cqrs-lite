package metaengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// CatchUpState is the observable outcome of [Store.CatchUpEngine] for one
// engine (the ADR-0137 write-failover recovery half). Surfaced via
// [Store.CatchUpSnapshot], the CatchUp field of [EngineStats], and the
// Doctor report, so an operator can see whether a quarantined engine's
// rebuild is in flight, failed (and why), or completed — instead of
// inferring it from quarantine state alone.
type CatchUpState struct {
	// Running is true while a catch-up rebuild is in progress.
	Running bool
	// LastError is the error text of the most recent failed rebuild; empty
	// when none and after a success.
	LastError string
	// Replayed is the number of events the last successful rebuild folded
	// into the engine (the log length at its stability gate).
	Replayed int
	// CompletedAt is when the last successful rebuild finished; zero when
	// the engine was never rebuilt.
	CompletedAt time.Time
}

// catchUpRecord is the mutable per-engine bookkeeping behind CatchUpState,
// guarded by healthMu (the ADR-0137 state mutex; a leaf — it never wraps
// s.mu or the EventLog lock).
type catchUpRecord struct {
	running   bool
	lastErr   string
	replayed  int
	completed time.Time
}

// catchUpStateUpdate applies fn to the engine's record under healthMu,
// creating the record on first use.
func (s *Store) catchUpStateUpdate(name string, fn func(rec *catchUpRecord)) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	if s.catchUps == nil {
		s.catchUps = make(map[string]*catchUpRecord)
	}

	rec := s.catchUps[name]
	if rec == nil {
		rec = &catchUpRecord{}
		s.catchUps[name] = rec
	}

	fn(rec)
}

// CatchUpSnapshot returns the current catch-up state keyed by engine name.
// Engines never rebuilt are absent — a missing entry means "no rebuild has
// been attempted", which is the normal state of every healthy store.
func (s *Store) CatchUpSnapshot() map[string]CatchUpState {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()

	out := make(map[string]CatchUpState, len(s.catchUps))
	for name, rec := range s.catchUps {
		out[name] = CatchUpState{
			Running:     rec.running,
			LastError:   rec.lastErr,
			Replayed:    rec.replayed,
			CompletedAt: rec.completed,
		}
	}

	return out
}

// doctorCatchUpSection renders the catch-up lines for the Doctor report.
// Returns the empty string when no rebuild was ever attempted.
func (s *Store) doctorCatchUpSection() string {
	snap := s.CatchUpSnapshot()
	if len(snap) == 0 {
		return ""
	}

	var b strings.Builder

	for _, name := range slices.Sorted(maps.Keys(snap)) {
		st := snap[name]

		switch {
		case st.Running:
			fmt.Fprintf(&b, "  %s: catch-up running\n", name)

		case st.LastError != "":
			fmt.Fprintf(&b, "  %s: catch-up FAILED (last error: %s)\n", name, st.LastError)

		default:
			fmt.Fprintf(&b, "  %s: caught up (replayed %d events at %s)\n",
				name, st.Replayed, st.CompletedAt.Format(time.RFC3339))
		}
	}

	return b.String()
}

// writeDoctorCatchUpSection appends the Catch-Up section (header + lines) to
// the Doctor report; a no-op while no engine was ever rebuilt.
func (s *Store) writeDoctorCatchUpSection(b *strings.Builder) {
	b.WriteString(s.doctorCatchUpSectionFull())
}

// doctorCatchUpSectionFull renders the whole section including its header,
// empty while there is nothing to report.
func (s *Store) doctorCatchUpSectionFull() string {
	section := s.doctorCatchUpSection()
	if section == "" {
		return ""
	}

	return "\n--- Catch-Up ---\n" + section
}
