package testhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// TimelineEvent describes an event in a timeline test.
type TimelineEvent struct {
	Type    string
	Version event.Version
	Offset  time.Duration
}

// MakeTimelineEvents creates events with timestamps relative to now.
// Use negative offsets for past events (e.g., -2*time.Hour for 2 hours ago).
func MakeTimelineEvents(
	tb testing.TB,
	aggType event.AggregateType,
	aggID id.AggregateID,
	events []TimelineEvent,
) []event.Event {
	tb.Helper()

	now := time.Now()
	result := make([]event.Event, len(events))
	for i, e := range events {
		evt, err := event.NewEvent(
			event.Type(e.Type),
			aggID,
			aggType,
			e.Version,
			nil,
			event.WithOccurredAt(now.Add(e.Offset)),
		)
		if err != nil {
			tb.Fatalf("MakeTimelineEvents: create event %q: %v", e.Type, err)
		}
		result[i] = evt
	}

	return result
}

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

// AppendBatcher is the minimal interface needed for MakeLoadToTimestampFixtures.
type AppendBatcher interface {
	AppendBatch(ctx context.Context, aggType event.AggregateType, aggID id.AggregateID, events []event.Event) error
}

// MakeLoadToTimestampFixtures creates a standard 3-event timeline (Created, Updated, Deleted)
// and appends them to the store. Returns the current time and the aggID used.
func MakeLoadToTimestampFixtures(
	tb testing.TB,
	store AppendBatcher,
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
	versions [3]event.Version,
) (time.Time, id.AggregateID) {
	tb.Helper()

	now := time.Now()
	events := MakeTimelineEvents(tb, aggType, aggID, []TimelineEvent{
		{Type: "Created", Version: versions[0], Offset: -2 * time.Hour},
		{Type: "Updated", Version: versions[1], Offset: -1 * time.Hour},
		{Type: "Deleted", Version: versions[2], Offset: 0},
	})

	if err := store.AppendBatch(ctx, aggType, aggID, events); err != nil {
		tb.Fatalf("AppendBatch: %v", err)
	}

	return now, aggID
}

// QuickSnapshot creates a snapshot with the given parameters.
func QuickSnapshot(
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	state []byte,
) event.Snapshot {
	return event.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         state,
	}
}
