package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

const eventKeyPrefix = "evt:"

// EventStore implements event.Store using a Backend.
// Key scheme: "evt:{aggregateType}:{aggregateID}:{version:010d}".
type EventStore struct {
	backend Backend
}

// NewEventStore creates an event.Store backed by any store.Backend.
func NewEventStore(backend Backend) *EventStore {
	return &EventStore{backend: backend}
}

func eventKey(ref event.AggregateRef, version event.Version) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%010d", eventKeyPrefix, ref.Type, ref.ID, version.Int())
}

func eventPrefix(ref event.AggregateRef) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", eventKeyPrefix, ref.Type, ref.ID)
}

func (s *EventStore) Save(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	return s.backend.Batch(context.Background(), func(tx Transaction) error {
		if expectedVersion > 0 {
			lastKey := eventKey(ref, expectedVersion)
			if _, err := tx.Get(lastKey); err != nil {
				return event.WrapConflict(
					event.CheckVersionConflict(0, expectedVersion),
					"store.event_save",
					fmt.Sprintf("version check for %s: key %s not found", ref, string(lastKey)),
				)
			}
		}

		if expectedVersion == 0 {
			firstKey := eventKey(ref, 1)
			if _, err := tx.Get(firstKey); err == nil {
				return event.WrapConflict(
					event.CheckVersionConflict(1, 0),
					"store.event_save",
					fmt.Sprintf("version check for %s: aggregate already has events", ref),
				)
			}
		}

		for i, evt := range events {
			expected := event.Version(expectedVersion.Int() + i + 1)
			if evt.Version() != expected {
				return event.WrapConflict(
					fmt.Errorf("event version mismatch: expected %d, got %d", expected.Int(), evt.Version().Int()),
					"store.event_version",
					"version validation",
				)
			}

			data, err := marshalEvent(evt)
			if err != nil {
				return fmt.Errorf("marshal event: %w", err)
			}

			if err := tx.Put(eventKey(ref, evt.Version()), data); err != nil {
				return fmt.Errorf("put event: %w", err)
			}
		}

		return nil
	})
}

func (s *EventStore) AppendBatch(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	return s.backend.Batch(context.Background(), func(tx Transaction) error {
		for _, evt := range events {
			data, err := marshalEvent(evt)
			if err != nil {
				return fmt.Errorf("marshal event: %w", err)
			}

			if err := tx.Put(eventKey(ref, evt.Version()), data); err != nil {
				return fmt.Errorf("put event: %w", err)
			}
		}

		return nil
	})
}

func (s *EventStore) Load(
	_ context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	return s.loadFromPrefix(ref, nil)
}

func (s *EventStore) LoadFromVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	events, err := s.loadFromPrefix(ref, nil)
	if err != nil {
		return nil, err
	}

	var filtered []event.Event
	for _, evt := range events {
		if evt.Version() > version {
			filtered = append(filtered, evt)
		}
	}

	if len(filtered) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return filtered, nil
}

func (s *EventStore) LoadToVersion(
	_ context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	events, err := s.loadFromPrefix(ref, nil)
	if err != nil {
		return nil, err
	}

	var filtered []event.Event

	for _, evt := range events {
		if evt.Version() <= maxVersion {
			filtered = append(filtered, evt)
		}
	}

	if len(filtered) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return filtered, nil
}

func (s *EventStore) LoadToTimestamp(
	_ context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	events, err := s.loadFromPrefix(ref, nil)
	if err != nil {
		return nil, err
	}

	var filtered []event.Event

	for _, evt := range events {
		if !evt.OccurredAt().After(maxTime) {
			filtered = append(filtered, evt)
		}
	}

	if len(filtered) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return filtered, nil
}

func (s *EventStore) Close() error { return nil }

func (s *EventStore) loadFromPrefix(ref event.AggregateRef, _ []byte) ([]event.Event, error) {
	prefix := eventPrefix(ref)
	upper := fmt.Appendf(nil, "%s%s:%s:\xff", eventKeyPrefix, ref.Type, ref.ID)

	events, err := s.iterateRange(prefix, upper)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

func (s *EventStore) iterateRange(lower, upper []byte) ([]event.Event, error) {
	it, err := s.backend.Scan(context.Background(), lower)
	if err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}
	defer it.Close()

	var events []event.Event

	for it.Next() {
		evt, err := unmarshalEvent(it.Value())
		if err != nil {
			return nil, fmt.Errorf("unmarshal event at key %s: %w", string(it.Key()), err)
		}

		events = append(events, evt)
	}

	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return events, nil
}

var _ event.Store = (*EventStore)(nil)

type serializableEvent struct {
	ID            id.EventID      `json:"id"`
	Type          string          `json:"type"`
	AggregateID   id.AggregateID  `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int             `json:"version"`
	SchemaVersion int             `json:"schema_version,omitempty"`
	Payload       []byte          `json:"payload"`
	OccurredAt    int64           `json:"occurred_at"`
	Metadata      *event.Metadata `json:"metadata,omitempty"`
}

func marshalEvent(evt event.Event) ([]byte, error) {
	s := serializableEvent{
		ID:            evt.ID(),
		Type:          string(evt.Type()),
		AggregateID:   evt.AggregateID(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       evt.Payload(),
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Metadata:      evt.Metadata(),
	}

	return json.Marshal(s)
}

func unmarshalEvent(data []byte) (event.Event, error) {
	var s serializableEvent
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	opts := []event.Option{
		event.WithEventID(s.ID),
		event.WithOccurredAt(time.Unix(0, s.OccurredAt)),
	}
	if s.SchemaVersion > 0 {
		opts = append(opts, event.WithSchemaVersion(event.SchemaVersion(s.SchemaVersion)))
	}

	if s.Metadata != nil {
		opts = append(opts, event.WithMetadata(s.Metadata))
	}

	evt, err := event.NewEvent(
		event.Type(s.Type),
		s.AggregateID,
		event.AggregateType(s.AggregateType),
		event.Version(s.Version),
		s.Payload,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct event: %w", err)
	}

	return evt, nil
}

// Compile-time check that EventStore implements io.Closer.
var _ io.Closer = (*EventStore)(nil)
