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
