package testhelpers

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// NewTestEvent creates a test event with standard test values.
func NewTestEvent() (event.Event, error) {
	return MakeEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
}

// NewEventOpts creates an event with the given parameters and options, fataling on error.
func NewEventOpts(
	tb testing.TB,
	typ event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(typ, aggID, aggType, version, payload, opts...)
	if err != nil {
		tb.Fatalf("create event %q: %v", typ, err)
	}

	return evt
}

// NewEvent creates an event with the given parameters, fataling on error.
func NewEvent(
	t *testing.T,
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)
	if err != nil {
		t.Fatalf("create event %q: %v", eventType, err)
	}

	return evt
}

// MakeEvent creates an event with the given parameters, returning an error.
func MakeEvent(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) (event.Event, error) {
	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt, err //nolint:wrapcheck // thin wrapper, caller adds context if needed
}

// QuickEvent creates an event with the given parameters, discarding errors.
func QuickEvent(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) event.Event {
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt
}

// TamperEvent creates a copy of the original event with a different payload but
// the same ID, timestamp, schema version, and metadata — useful for testing
// tamper detection in signing middleware.
func TamperEvent(original event.Event, newPayload []byte) event.Event {
	tampered, _ := event.NewEvent(
		original.Type(),
		original.AggregateID(),
		original.AggregateType(),
		original.Version(),
		newPayload,
		event.WithEventID(original.ID()),
		event.WithOccurredAt(original.OccurredAt()),
		event.WithSchemaVersion(original.SchemaVersion()),
		event.WithMetadata(original.Metadata()),
	)

	return tampered
}

// QuickEventOpts creates an event with the given parameters and options, discarding errors.
func QuickEventOpts(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload, opts...)

	return evt
}
