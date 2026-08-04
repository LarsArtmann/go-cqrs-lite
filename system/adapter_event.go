package system

import (
	"context"
	"fmt"
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
type EventAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
}

// NewEventAdapter creates an event.Store backed by a StreamLogBackend.
// The collection parameter names the stream log collection (e.g., "events").
func NewEventAdapter(backend metaengine.StreamLogBackend, collection string) *EventAdapter {
	return &EventAdapter{
		backend:    backend,
		collection: collection,
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

	if tx, ok := a.backend.(metaengine.Transactional); ok {
		return tx.RunInTx(ctx, func(ctx context.Context) error {
			current, err := a.backend.StreamVersion(ctx, a.collection, sid)
			if err != nil {
				return fmt.Errorf("event adapter: stream version: %w", err)
			}

			if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
				return err
			}

			return a.backend.StreamAppend(ctx, a.collection, sid, eventsToAny(events))
		})
	}

	current, err := a.backend.StreamVersion(ctx, a.collection, sid)
	if err != nil {
		return fmt.Errorf("event adapter: stream version: %w", err)
	}

	if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
		return err
	}

	return a.backend.StreamAppend(ctx, a.collection, sid, eventsToAny(events))
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
	// The StreamLogBackend uses int64 sequence positions, not event IDs.
	// We find the seq of afterEventID by scanning, then read from there.
	all, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from: %w", err)
	}

	afterSeq := int64(0)

	for i, val := range all {
		evt, ok := val.(event.Event)
		if ok && evt.ID() == afterEventID {
			afterSeq = int64(i + 1) // skip the event itself

			break
		}
	}

	values, err := a.backend.JournalReadFrom(ctx, a.collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from: %w", err)
	}

	return anyToEvents(values)
}

// ─── helpers ─---

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
