package eventtest

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

const createdEventType = "Created"

type TimelineEvent struct {
	Type    string
	Version event.Version
	Offset  time.Duration
}

func MakeTimelineEvents(
	tb testing.TB,
	aggType id.StreamType,
	aggID id.StreamID,
	events []TimelineEvent,
) []event.Event {
	tb.Helper()

	now := time.Now()

	result := make([]event.Event, len(events))
	for i, e := range events {
		//cqrs-lint:ignore(A014,D011) library code or intentional pattern
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

func NewTestEvent() (event.Event, error) {
	return MakeEvent("test.evt", id.NewStreamID(), "Test", 1, nil)
}

func NewEventOpts(
	tb testing.TB,
	typ event.Type,
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	tb.Helper()

	//cqrs-lint:ignore(A014) library code or intentional pattern
	evt, err := event.NewEvent(typ, aggID, aggType, version, payload, opts...)
	if err != nil {
		tb.Fatalf("create event %q: %v", typ, err)
	}

	return evt
}

func NewEvent(
	t *testing.T,
	eventType event.Type,
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	payload []byte,
) event.Event {
	t.Helper()

	//cqrs-lint:ignore(A014) library code or intentional pattern
	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)
	if err != nil {
		t.Fatalf("create event %q: %v", eventType, err)
	}

	return evt
}

//cqrs-lint:ignore(B001) library code or intentional pattern
func MakeEvent(
	eventType event.Type,
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	payload []byte,
) (event.Event, error) {
	//cqrs-lint:ignore(A014) library code or intentional pattern
	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt, err //nolint:wrapcheck // thin wrapper, caller adds context if needed
}

func QuickEvent(
	eventType event.Type,
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	payload []byte,
) event.Event {
	//cqrs-lint:ignore(A014) library code or intentional pattern
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt
}

func QuickEventOpts(
	eventType event.Type,
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	//cqrs-lint:ignore(A014) library code or intentional pattern
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload, opts...)

	return evt
}

func TamperEvent(original event.Event, newPayload []byte) event.Event {
	//cqrs-lint:ignore(A014) library code or intentional pattern
	tampered, _ := event.NewEvent(
		original.Type(),
		original.StreamID(),
		original.StreamType(),
		original.Version(),
		newPayload,
		event.WithEventID(original.ID()),
		event.WithOccurredAt(original.OccurredAt()),
		event.WithSchemaVersion(original.SchemaVersion()),
		event.WithMetadata(original.Metadata()),
	)

	return tampered
}

type AppendBatcher interface {
	AppendBatch(
		ctx context.Context,
		ref id.StreamRef,
		events []event.Event,
	) error
}

func MakeLoadToTimestampFixtures(
	tb testing.TB,
	store AppendBatcher,
	ctx context.Context,
	aggType id.StreamType,
	aggID id.StreamID,
	versions [3]event.Version,
) (time.Time, id.StreamID) {
	tb.Helper()

	now := time.Now()
	eventTypeCreated := createdEventType
	events := MakeTimelineEvents(tb, aggType, aggID, []TimelineEvent{
		{Type: eventTypeCreated, Version: versions[0], Offset: -2 * time.Hour},
		{Type: "Updated", Version: versions[1], Offset: -1 * time.Hour},
		{Type: "Deleted", Version: versions[2], Offset: 0},
	})

	err := store.AppendBatch(ctx, id.NewStreamRef(aggType, aggID), events)
	if err != nil {
		tb.Fatalf("AppendBatch: %v", err)
	}

	return now, aggID
}

func MakeThreeTimelineEvents(
	tb testing.TB,
	aggType1 id.StreamType,
	aggID1 id.StreamID,
	aggType2 id.StreamType,
	aggID2 id.StreamID,
) ([]event.Event, []event.Event, []event.Event) {
	tb.Helper()

	evt1 := MakeTimelineEvents(tb, aggType1, aggID1, []TimelineEvent{
		{Type: createdEventType, Version: 1, Offset: -2 * time.Hour},
	})
	evt2 := MakeTimelineEvents(tb, aggType2, aggID2, []TimelineEvent{
		{Type: createdEventType, Version: 1, Offset: -1 * time.Hour},
	})
	evt3 := MakeTimelineEvents(tb, aggType1, aggID1, []TimelineEvent{
		{Type: "Updated", Version: 1, Offset: 0},
	})

	return evt1, evt2, evt3
}

func QuickSnapshot(
	aggID id.StreamID,
	aggType id.StreamType,
	version event.Version,
	state []byte,
) snapshot.Snapshot {
	return snapshot.Snapshot{
		StreamID:   aggID,
		StreamType: aggType,
		Version:    version,
		State:      state,
		CreatedAt:  time.Now(),
	}
}
