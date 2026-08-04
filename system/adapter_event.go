package system

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EventAdapter wraps a [metaengine.StreamLogBackend] as an [event.Store].
// It bridges the new storage primitive (stream-keyed append-only log of any
// values) to the standard CQRS event interfaces (EventSink, EventSource,
// Journal, SeekableJournal).
//
// For the Memory engine, events are stored as direct pointers (no encoding).
// For SQL engines, events are encoded via a codec at the adapter boundary.
//
// Optimistic concurrency uses [metaengine.AtomicAppender] when available
// (single-lock atomic version-check-then-append), falling back to
// [metaengine.Transactional.RunInTx], then to a non-atomic check-then-append.
type EventAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string

	// seqCache maps event ID strings to global journal sequence numbers.
	// Built lazily on the first ReadFrom call; updated incrementally from
	// ReadFrom results. Enables O(1) EventID→seq lookup instead of O(N) scan.
	seqCache   map[string]int64
	seqCacheMu sync.RWMutex
}

// NewEventAdapter creates an event.Store backed by a StreamLogBackend.
// The collection parameter names the stream log collection (e.g., "events").
func NewEventAdapter(backend metaengine.StreamLogBackend, collection string) *EventAdapter {
	return &EventAdapter{
		backend:    backend,
		collection: collection,
		seqCache:   make(map[string]int64),
	}
}

// Compile-time assertions.
var (
	_ event.Store           = (*EventAdapter)(nil)
	_ event.Journal         = (*EventAdapter)(nil)
	_ event.SeekableJournal = (*EventAdapter)(nil)
)

// ─── EventSink ───

func (a *EventAdapter) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	sid := ref.StreamKey()
	values := eventsToAny(events)

	// Fast path: engine supports atomic version-check-then-append.
	if ap, ok := a.backend.(metaengine.AtomicAppender); ok {
		if err := ap.StreamAppendExpected(ctx, a.collection, sid, int64(expectedVersion), values); err != nil {
			if errors.Is(err, metaengine.ErrVersionConflict) {
				return event.ErrVersionConflict
			}

			return fmt.Errorf("event adapter: save: %w", err)
		}

		return nil
	}

	// Fallback: use Transactional if available.
	if tx, ok := a.backend.(metaengine.Transactional); ok {
		return tx.RunInTx(ctx, func(ctx context.Context) error {
			current, err := a.backend.StreamVersion(ctx, a.collection, sid)
			if err != nil {
				return fmt.Errorf("event adapter: stream version: %w", err)
			}

			if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
				return err
			}

			return a.backend.StreamAppend(ctx, a.collection, sid, values)
		})
	}

	// Last resort: non-atomic check-then-append.
	current, err := a.backend.StreamVersion(ctx, a.collection, sid)
	if err != nil {
		return fmt.Errorf("event adapter: stream version: %w", err)
	}

	if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
		return err
	}

	return a.backend.StreamAppend(ctx, a.collection, sid, values)
}

func (a *EventAdapter) AppendBatch(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	sid := ref.StreamKey()

	return a.backend.StreamAppend(ctx, a.collection, sid, eventsToAny(events))
}

// ─── EventSource ───

func (a *EventAdapter) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	sid := ref.StreamKey()

	values, err := a.backend.StreamRead(ctx, a.collection, sid)
	if err != nil {
		return nil, fmt.Errorf("event adapter: load: %w", err)
	}

	return anyToEvents(values)
}

func (a *EventAdapter) LoadFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) ([]event.Event, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	start := min(int(version), len(all))

	return all[start:], nil
}

func (a *EventAdapter) LoadToVersion(
	ctx context.Context,
	ref id.StreamRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	end := min(int(maxVersion), len(all))

	return all[:end], nil
}

func (a *EventAdapter) LoadToTimestamp(
	ctx context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]event.Event, error) {
	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	result := make([]event.Event, 0, len(all))
	for _, evt := range all {
		if !evt.OccurredAt().After(maxTime) {
			result = append(result, evt)
		}
	}

	return result, nil
}

// ─── Journal ───

func (a *EventAdapter) ReadAll(ctx context.Context) ([]event.Event, error) {
	values, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read all: %w", err)
	}

	return anyToEvents(values)
}

// ─── SeekableJournal ───

func (a *EventAdapter) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	afterSeq := a.lookupSeq(ctx, afterEventID)

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from: %w", err)
	}

	events, err := anyToEvents(values)
	if err != nil {
		return nil, err
	}

	// Incrementally populate the seq cache from results.
	// Journal entries are contiguous: the i-th entry after afterSeq has
	// seq = afterSeq + i + 1.
	a.seqCacheMu.Lock()
	for i, evt := range events {
		a.seqCache[evt.ID().String()] = afterSeq + int64(i) + 1
	}
	a.seqCacheMu.Unlock()

	return events, nil
}

// lookupSeq returns the global journal sequence number for the given event ID.
// It checks the seq cache first (O(1)); on miss, it scans the full journal
// once to build the cache, then retries. Subsequent lookups for events in the
// cache are O(1).
func (a *EventAdapter) lookupSeq(ctx context.Context, eventID id.EventID) int64 {
	key := eventID.String()

	a.seqCacheMu.RLock()
	seq, ok := a.seqCache[key]
	a.seqCacheMu.RUnlock()

	if ok {
		return seq
	}

	// Cache miss: scan the journal to populate the cache.
	all, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return 0
	}

	a.seqCacheMu.Lock()
	defer a.seqCacheMu.Unlock()

	for i, val := range all {
		if evt, ok := val.(event.Event); ok {
			a.seqCache[evt.ID().String()] = int64(i + 1)
		}
	}

	if seq, ok := a.seqCache[key]; ok {
		return seq
	}

	return 0
}

// ─── helpers ───

func eventsToAny(events []event.Event) []any {
	result := make([]any, len(events))
	for i, evt := range events {
		result[i] = evt
	}

	return result
}

func anyToEvents(values []any) ([]event.Event, error) {
	result := make([]event.Event, 0, len(values))
	for _, val := range values {
		evt, ok := val.(event.Event)
		if !ok {
			return nil, fmt.Errorf("event adapter: value is not event.Event (got %T)", val)
		}

		result = append(result, evt)
	}

	return result, nil
}
