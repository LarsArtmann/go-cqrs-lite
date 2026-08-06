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

// EventAdapterOption tunes an EventAdapter at construction time.
type EventAdapterOption func(*EventAdapter)

// WithSerialization enables event serialization for persistent engines
// (SQLite, Pebble). When enabled, events are encoded to JSON envelope strings
// on write and decoded on read. For the Memory engine, this option should NOT
// be set — events are stored as direct pointers.
func WithSerialization() EventAdapterOption {
	return func(a *EventAdapter) { a.serialize = true }
}

// EventAdapter wraps a [metaengine.StreamLogBackend] as an [event.Store].
type EventAdapter struct {
	backend    metaengine.StreamLogBackend
	collection string
	serialize  bool

	// temporal is non-nil when the backend implements StreamTemporalReader,
	// enabling efficient version-bounded reads without loading the full stream.
	temporal metaengine.StreamTemporalReader

	seqCache   map[string]int64
	seqCacheMu sync.RWMutex
}

// NewEventAdapter creates an event.Store backed by a StreamLogBackend.
func NewEventAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...EventAdapterOption,
) *EventAdapter {
	a := &EventAdapter{
		backend:    backend,
		collection: collection,
		seqCache:   make(map[string]int64),
	}

	for _, opt := range opts {
		opt(a)
	}

	// Detect StreamTemporalReader support for efficient version-bounded reads.
	if tr, ok := backend.(metaengine.StreamTemporalReader); ok {
		a.temporal = tr
	}

	return a
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
	values := a.eventsToAny(events)

	if ap, ok := a.backend.(metaengine.AtomicAppender); ok {
		if err := ap.StreamAppendExpected(
			ctx,
			a.collection,
			sid,
			int64(expectedVersion),
			values,
		); err != nil {
			if errors.Is(err, metaengine.ErrVersionConflict) {
				return event.ErrVersionConflict
			}

			return fmt.Errorf("event adapter: save: %w", err)
		}

		return nil
	}

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
	return a.backend.StreamAppend(ctx, a.collection, ref.StreamKey(), a.eventsToAny(events))
}

// ─── EventSource ───

func (a *EventAdapter) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	values, err := a.backend.StreamRead(ctx, a.collection, ref.StreamKey())
	if err != nil {
		return nil, fmt.Errorf("event adapter: load: %w", err)
	}

	return a.anyToEvents(values)
}

// loadVersioned implements the temporal-fast-path-then-fallback pattern shared
// by LoadFromVersion and LoadToVersion. temporalRead is only called when
// a.temporal is non-nil. sliceFallback slices the full-stream fallback result.
func (a *EventAdapter) loadVersioned(
	ctx context.Context,
	ref id.StreamRef,
	temporalRead func() ([]any, error),
	errLabel string,
	sliceFallback func(all []event.Event) []event.Event,
) ([]event.Event, error) {
	if a.temporal != nil {
		values, err := temporalRead()
		if err != nil {
			return nil, fmt.Errorf("event adapter: %s: %w", errLabel, err)
		}

		return a.anyToEvents(values)
	}

	all, err := a.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return sliceFallback(all), nil
}

func (a *EventAdapter) LoadFromVersion(
	ctx context.Context, ref id.StreamRef, version event.Version,
) ([]event.Event, error) {
	// StreamReadFromVersion is 1-indexed inclusive; LoadFromVersion's version
	// is 0-indexed (skip N events), so add 1.
	return a.loadVersioned(
		ctx, ref,
		func() ([]any, error) {
			return a.temporal.StreamReadFromVersion(
				ctx, a.collection, ref.StreamKey(), int64(version)+1,
			)
		},
		"load from version",
		func(all []event.Event) []event.Event {
			return all[min(int(version), len(all)):]
		},
	)
}

func (a *EventAdapter) LoadToVersion(
	ctx context.Context, ref id.StreamRef, maxVersion event.Version,
) ([]event.Event, error) {
	return a.loadVersioned(
		ctx, ref,
		func() ([]any, error) {
			return a.temporal.StreamReadAsOfVersion(
				ctx, a.collection, ref.StreamKey(), int64(maxVersion),
			)
		},
		"load to version",
		func(all []event.Event) []event.Event {
			return all[:min(int(maxVersion), len(all))]
		},
	)
}

func (a *EventAdapter) LoadToTimestamp(
	ctx context.Context, ref id.StreamRef, maxTime time.Time,
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

	return a.anyToEvents(values)
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

	events, err := a.anyToEvents(values)
	if err != nil {
		return nil, err
	}

	a.seqCacheMu.Lock()
	for i, evt := range events {
		a.seqCache[evt.ID().String()] = afterSeq + int64(i) + 1
	}
	a.seqCacheMu.Unlock()

	return events, nil
}

// lookupSeq returns the global journal sequence number for the given event ID.
func (a *EventAdapter) lookupSeq(ctx context.Context, eventID id.EventID) int64 {
	key := eventID.String()

	a.seqCacheMu.RLock()
	seq, ok := a.seqCache[key]
	a.seqCacheMu.RUnlock()

	if ok {
		return seq
	}

	all, err := a.backend.JournalReadAll(ctx, a.collection)
	if err != nil {
		return 0
	}

	events, _ := a.anyToEvents(all)

	a.seqCacheMu.Lock()
	defer a.seqCacheMu.Unlock()

	for i, evt := range events {
		a.seqCache[evt.ID().String()] = int64(i + 1)
	}

	if seq, ok := a.seqCache[key]; ok {
		return seq
	}

	return 0
}


