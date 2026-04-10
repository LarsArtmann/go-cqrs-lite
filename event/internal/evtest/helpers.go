package evtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// NewTestEvent creates a new test event with default values.
func NewTestEvent(t testing.TB, eventType event.Type) event.Event {
	t.Helper()

	aggregateID := id.NewAggregateID()

	evt, err := event.NewEvent(eventType, aggregateID, "Test", 0, nil)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

// NewTestEventWithAggregate creates a new test event with specific aggregate.
func NewTestEventWithAggregate(
	t testing.TB,
	eventType event.Type,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	version int,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(eventType, aggregateID, aggregateType, version, nil)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

// NewBus creates a new memory bus and context for testing.
func NewBus(t testing.TB) (*event.MemoryBus, context.Context) {
	t.Helper()

	return event.NewMemoryBus(), context.Background()
}

// NewStore creates a new memory store and context for testing.
func NewStore(t testing.TB) (*event.MemoryStore, context.Context) {
	t.Helper()

	return event.NewMemoryStore(), context.Background()
}

// CollectingHandler returns a handler that appends events to the provided slice.
func CollectingHandler(t testing.TB, received *[]event.Event) event.Handler {
	t.Helper()

	return func(_ context.Context, evt event.Event) error {
		*received = append(*received, evt)

		return nil
	}
}

// ErrorHandler returns a handler that returns the given error.
func ErrorHandler(t testing.TB, err error) event.Handler {
	t.Helper()

	return func(_ context.Context, _ event.Event) error {
		return err
	}
}

// NoopHandler returns a handler that does nothing.
func NoopHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

// CallbackHandler returns a handler that calls the provided function.
func CallbackHandler(fn func()) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		fn()
		return nil
	}
}

// Context returns a background context for testing.
func Context(t testing.TB) context.Context {
	t.Helper()

	return context.Background()
}

// NewAggregateID creates a new aggregate ID for testing.
func NewAggregateID(t testing.TB) id.AggregateID {
	t.Helper()

	return id.NewAggregateID()
}

// NewEventID creates a new event ID for testing.
func NewEventID(t testing.TB) id.EventID {
	t.Helper()

	return id.NewEventID()
}

// GenerateUUID generates a new UUID string.
func GenerateUUID() string {
	return uuid.New().String()
}
