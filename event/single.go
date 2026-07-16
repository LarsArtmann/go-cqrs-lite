package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Single creates a single event and returns it as a one-element slice.
// This is a convenience wrapper around New for the common case where a
// decide function emits exactly one event.
//
// Example:
//
//	func decideCreate(s State, cmd CreateCmd) ([]event.Event, error) {
//	    return event.Single("user.created", cmd.AggregateID(), "User", s.Version.Increment(), UserCreated{Name: cmd.Name})
//	}
//
// This replaces the singleEvent/makeEvent/mustEvent helper functions
// that every consumer project reimplements.
func Single(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
	version Version,
	payload any,
	opts ...Option,
) ([]Event, error) {
	evt, err := New(eventType, aggregateID, aggregateType, version, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("event.Single: %w", err)
	}

	return []Event{evt}, nil
}
