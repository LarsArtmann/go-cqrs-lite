package metaengine

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// isSyntheticRecord reports whether rec carries no per-instance context:
// only the Type is set (the form Store.Apply synthesizes). Event-sourced
// records always carry a stream reference and a 1-indexed version, so an
// empty StreamID plus zero Version identifies the synthesized form.
func isSyntheticRecord(rec record.Record) bool {
	return rec.StreamID == "" && rec.Version == 0
}

// feedEntryPoint names the public Store method that fed an event into the
// apply path. It exists so the Doctor's synthetic-Record breakdown can point
// at the exact caller to fix instead of a single undifferentiated total.
type feedEntryPoint string

const (
	feedApply              feedEntryPoint = "Apply"
	feedApplyBatch         feedEntryPoint = "ApplyBatch"
	feedApplyIdempotent    feedEntryPoint = "ApplyIdempotent"
	feedApplyRecord        feedEntryPoint = "ApplyRecord"
	feedApplyEncoded       feedEntryPoint = "ApplyEncoded"
	feedApplyEncodedRecord feedEntryPoint = "ApplyEncodedRecord"
)

// syntheticFeedCounters breaks the synthetic-Record advisory count down by
// the entry point that fed it. Replays (Backfill/Verify/Demote/replication)
// never count — only direct applies do.
type syntheticFeedCounters struct {
	apply               atomic.Uint64
	applyBatch          atomic.Uint64
	applyIdempotent     atomic.Uint64
	applyRecord         atomic.Uint64
	applyEncoded        atomic.Uint64
	applyEncodedRecord  atomic.Uint64
}

// total sums every bucket — the pre-breakdown aggregate count.
func (c *syntheticFeedCounters) total() uint64 {
	return c.apply.Load() + c.applyBatch.Load() + c.applyIdempotent.Load() +
		c.applyRecord.Load() + c.applyEncoded.Load() + c.applyEncodedRecord.Load()
}

// bucket returns the counter for the given entry point.
func (c *syntheticFeedCounters) bucket(entry feedEntryPoint) *atomic.Uint64 {
	switch entry {
	case feedApplyBatch:
		return &c.applyBatch
	case feedApplyIdempotent:
		return &c.applyIdempotent
	case feedApplyRecord:
		return &c.applyRecord
	case feedApplyEncoded:
		return &c.applyEncoded
	case feedApplyEncodedRecord:
		return &c.applyEncodedRecord
	default:
		return &c.apply
	}
}

// breakdown renders the non-zero buckets as "Name=count" pairs joined by
// ", ", sorted by entry-point name for stable Doctor output.
func (c *syntheticFeedCounters) breakdown() string {
	parts := make([]string, 0, 6)

	for _, entry := range []feedEntryPoint{
		feedApply, feedApplyBatch, feedApplyEncoded,
		feedApplyEncodedRecord, feedApplyIdempotent, feedApplyRecord,
	} {
		if n := c.bucket(entry).Load(); n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", entry, n))
		}
	}

	return strings.Join(parts, ", ")
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

// recordAwareEventTypesCached is recordAwareEventTypes memoized: the map is
// computed on first use and invalidated by RegisterQuery when a runtime-
// registered query may add OnRecord folds. The apply hot path reads it under
// s.mu.RLock and RegisterQuery stores nil under the write lock, so the
// invalidate/recompute pair cannot interleave.
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
// it, attributed to the entry point that fed it. The first occurrence is
// logged (when a Logger is configured); every occurrence is counted for the
// Doctor's "--- Record context ---" section, broken down by entry point.
// The caller must hold s.mu (at least RLock).
func (s *Store) noteSyntheticRecordApply(entry feedEntryPoint, eventType string) {
	if !s.recordAwareEventTypesCached()[eventType] {
		return
	}

	s.syntheticFeeds.bucket(entry).Add(1)

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

	if applies := s.syntheticFeeds.total(); applies > 0 {
		fmt.Fprintf(
			&b,
			"  %d apply event(s) arrived with a synthesized Type-only Record — "+
				"OnRecord handlers saw empty StreamID/Version. "+
				"Use Store.ApplyRecord (or the projection adapter path) for full context.\n",
			applies,
		)
		fmt.Fprintf(&b, "  by entry point: %s\n", s.syntheticFeeds.breakdown())
	} else {
		b.WriteString("  all applies carried full Record context\n")
	}

	return b.String()
}
