package metaengine

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// EventLog records all applied events for consistency checking and replay.
type EventLog struct {
	mu     sync.Mutex
	events []EventInput
}

func NewEventLog() *EventLog { return &EventLog{} }

// EventInput pairs an event type with its payload for batch application.
// Record optionally carries the full record context: when set, ApplyBatch and
// the replay paths (Backfill, Verify, DemoteEngine catch-up) hand Record-aware
// projections the original StreamID/Version/metadata instead of a synthesized
// minimal record. Payload may also be the raw JSON bytes produced by
// ApplyEncoded/ApplyEncodedRecord (stored as jsontext.Value in the EventLog);
// every dispatch path decodes them per fold before invoke.
type EventInput struct {
	Type    string
	Payload any
	Record  record.Record
}

func (l *EventLog) Record(eventType string, payload any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, EventInput{Type: eventType, Payload: payload})
}

// RecordEvent records an event with its full record context so replay paths
// (Backfill, Verify, DemoteEngine catch-up) can rebuild Record-aware
// projections faithfully instead of from a synthesized minimal record.
func (l *EventLog) RecordEvent(eventType string, rec record.Record, payload any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, EventInput{Type: eventType, Payload: payload, Record: rec})
}

func (l *EventLog) Events() []EventInput {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]EventInput(nil), l.events...)
}

func (l *EventLog) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.events)
}

func (l *EventLog) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = nil
}

// eventsFrom returns a copy of the events recorded at and after offset,
// empty when the log holds nothing new. Replay paths use it to drain only
// the suffix that arrived since their previous pass.
func (l *EventLog) eventsFrom(offset int) []EventInput {
	l.mu.Lock()
	defer l.mu.Unlock()

	if offset >= len(l.events) {
		return nil
	}

	return append([]EventInput(nil), l.events[offset:]...)
}

// reactivateIfStable is the catch-up gate: holding the append lock, it
// reports whether events arrived since offset (grew) and, when none did,
// runs reactivate in the SAME critical section — so a quarantined engine is
// trusted again at a point where every recorded event is either already
// replayed or will be live-folded after reactivation. reactivate must call
// leaf code only (health transitions, metrics): the live apply path holds
// s.mu while recording (s.mu → l.mu), so acquiring s.mu here would invert
// the order and deadlock.
func (l *EventLog) reactivateIfStable(offset int, reactivate func()) (grew bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.events) != offset {
		return true
	}

	reactivate()

	return false
}

// WithEventLog attaches an event log to the Store.
func WithEventLog(store *Store, log *EventLog) {
	store.eventLog = log
}

// Verify replays all recorded events into a verification engine and compares
// the row counts against the live store. Returns nil if consistent.
// Requires WithEventLog and the original query declarations stored on Plan.
func (s *Store) Verify(ctx context.Context, engines []Engine) error {
	if s.eventLog == nil {
		return errNoEventLog
	}

	events := s.eventLog.Events()
	if len(events) == 0 {
		return nil
	}

	if s.queryDecls == nil {
		return errNoQueryDecls
	}

	freshStore, err := Plan(engines, s.queryDecls...)
	if err != nil {
		return fmt.Errorf("metaengine.Verify: plan fresh store: %w", err)
	}

	for _, evt := range events {
		if evt.Record.Type != "" {
			if err := freshStore.ApplyRecord(ctx, evt.Record, evt.Payload); err != nil {
				return fmt.Errorf("metaengine.Verify: replay %s: %w", evt.Type, err)
			}

			continue
		}

		if err := freshStore.Apply(ctx, evt.Type, evt.Payload); err != nil {
			return fmt.Errorf("metaengine.Verify: replay %s: %w", evt.Type, err)
		}
	}

	liveCols := s.Collections()
	freshCols := freshStore.Collections()

	for i := range liveCols {
		if i >= len(freshCols) {
			return errCollectionCountMismatch
		}

		liveEng, _ := s.collectionEngine(liveCols[i].Name)
		freshEng2, _ := freshStore.collectionEngine(freshCols[i].Name)

		liveCount := countRows(ctx, liveEng, liveCols[i].Name)
		freshCount := countRows(ctx, freshEng2, freshCols[i].Name)

		if liveCount != freshCount {
			return fmt.Errorf(
				"%w in %q — live has %d rows, replay has %d",
				errVerifyDrift, liveCols[i].Name, liveCount, freshCount,
			)
		}
	}

	return nil
}

func countRows(ctx context.Context, eng Engine, collection string) int {
	if sb, ok := eng.(ScanBackend); ok {
		result, err := sb.MapScan(ctx, collection, nil, nil, nil, 0)
		if err != nil {
			return -1
		}

		return len(result.Items)
	}

	return -1
}
