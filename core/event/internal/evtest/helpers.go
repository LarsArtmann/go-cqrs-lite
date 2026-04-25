package evtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// NewTestEvent creates a new test event with default values.
//nolint:ireturn // test helper returns interface by design
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
//nolint:ireturn // test helper returns interface by design
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
