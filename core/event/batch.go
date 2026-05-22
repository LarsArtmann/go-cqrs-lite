package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// NewEvents creates multiple events in batch with typed payloads.
// The version increments for each event. All events share the same aggregateID,
// aggregateType, and options but have different eventTypes and payloads.
func NewEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	eventTypes []Type,
	payloads []any,
	opts ...Option,
) ([]Event, error) {
	if len(eventTypes) != len(payloads) {
		return nil, fmt.Errorf(
			"%w: got %d event types and %d payloads",
			ErrMismatchedEventCount,
			len(eventTypes),
			len(payloads),
		)
	}

	if len(eventTypes) == 0 {
		return nil, nil
	}

	events := make([]Event, len(eventTypes))

	for i := range eventTypes {
		evtVersion := version.Add(i + 1)
		payload, err := marshalPayload(payloads[i], eventTypes[i])
		if err != nil {
			return nil, fmt.Errorf("marshal payload for event %d: %w", i, err)
		}

		evt, err := NewEvent(
			eventTypes[i],
			aggregateID,
			aggregateType,
			evtVersion,
			payload,
			opts...,
		)
		if err != nil {
			return nil, fmt.Errorf("create event %d: %w", i, err)
		}

		events[i] = evt
	}

	return events, nil
}

// MustNewEvents creates multiple events in batch, panicking on error.
// See NewEvents for details.
func MustNewEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	eventTypes []Type,
	payloads []any,
	opts ...Option,
) []Event {
	events, err := NewEvents(aggregateID, aggregateType, version, eventTypes, payloads, opts...)
	if err != nil {
		panic(err)
	}

	return events
}
