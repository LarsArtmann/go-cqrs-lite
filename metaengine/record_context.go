package metaengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// isSyntheticRecord reports whether rec carries no per-instance context:
// only the Type is set (the form Store.Apply synthesizes). Event-sourced
// records always carry a stream reference and a 1-indexed version, so an
// empty StreamID plus zero Version identifies the synthesized form.
func isSyntheticRecord(rec record.Record) bool {
	return rec.StreamID == "" && rec.Version == 0
}

// recordAwareEventTypes returns the event types whose registered folds were
// created via OnRecord/OnRecordTyped. The caller must hold s.mu (at least
// RLock); the result is freshly computed — most callers want
// recordAwareEventTypesCached.
func (s *Store) recordAwareEventTypes() map[string]bool {
	out := make(map[string]bool)

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		for _, f := range s.queries[name].QueryFolds() {
			if foldWantsRecord(f) {
				out[f.EventType()] = true
			}
		}
	}

	return out
}

// recordAwareEventTypesCached is recordAwareEventTypes memoized: folds are
// fixed at Plan time, so the map is computed once and read lock-free on the
// apply hot path.
func (s *Store) recordAwareEventTypesCached() map[string]bool {
	if p := s.recordAwareEvents.Load(); p != nil {
		return *p
	}

	m := s.recordAwareEventTypes()
	s.recordAwareEvents.CompareAndSwap(nil, &m)

	if p := s.recordAwareEvents.Load(); p != nil {
		return *p
	}

	return m
}

// noteSyntheticRecordApply records that an event reached the store as a
// synthesized Type-only Record while record-aware folds are registered for
// it. The first occurrence is logged (when a Logger is configured); every
// occurrence is counted for the Doctor's "--- Record context ---" section.
// The caller must hold s.mu (at least RLock).
func (s *Store) noteSyntheticRecordApply(eventType string) {
	if !s.recordAwareEventTypesCached()[eventType] {
		return
	}

	s.syntheticRecordApplies.Add(1)

	s.syntheticRecordAdvisory.Do(func() {
		logSyntheticRecordAdvisory(s.hooks, eventType)
	})
}

// logSyntheticRecordAdvisory emits the one-time warning through the store's
// configured logger, if any. With no logger the advisory stays silent here
// and surfaces through Doctor instead.
func logSyntheticRecordAdvisory(hooks *Hooks, eventType string) {
	if hooks == nil || hooks.Logger == nil {
		return
	}

	hooks.Logger.Printf(
		"[metaengine] event %q applied via Store.Apply with a Type-only Record: "+
			"OnRecord folds for it receive empty StreamID/Version — "+
			"use Store.ApplyRecord for full context",
		eventType,
	)
}

// recordContextDoctorSection renders the "--- Record context ---" section of
// the Doctor() report. It lists the event types whose OnRecord folds expect
// Record context and counts applies that arrived without it, making the
// silent zero-Record hazard visible at runtime.
func (s *Store) recordContextDoctorSection() string {
	s.mu.RLock()
	aware := s.recordAwareEventTypes()
	s.mu.RUnlock()

	if len(aware) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n--- Record context ---\n")
	fmt.Fprintf(
		&b,
		"  %d record-aware event type(s): %s\n",
		len(aware),
		strings.Join(slices.Sorted(maps.Keys(aware)), ", "),
	)

	if applies := s.syntheticRecordApplies.Load(); applies > 0 {
		fmt.Fprintf(
			&b,
			"  %d apply event(s) arrived with a synthesized Type-only Record (Store.Apply) — "+
				"OnRecord handlers saw empty StreamID/Version. "+
				"Use Store.ApplyRecord (or the projection adapter path) for full context.\n",
			applies,
		)
	} else {
		b.WriteString("  all applies carried full Record context\n")
	}

	return b.String()
}
