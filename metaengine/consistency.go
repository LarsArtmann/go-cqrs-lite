package metaengine

import (
	"context"
	"fmt"
	"sync"
)

// EventLog records all applied events for consistency checking and replay.
type EventLog struct {
	mu     sync.Mutex
	events []EventInput
}

func NewEventLog() *EventLog { return &EventLog{} }

func (l *EventLog) Record(eventType string, payload any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, EventInput{Type: eventType, Payload: payload})
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
