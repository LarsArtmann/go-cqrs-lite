package event

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
		return nil, Wrap(
			ErrMismatchedEventCount,
			Rejection,
			"event.mismatched_event_count",
			"event types and payloads count must match: got "+
				strconv.Itoa(len(eventTypes))+" event types and "+
				strconv.Itoa(len(payloads))+" payloads",
		)
	}

	if len(eventTypes) == 0 {
		return nil, nil
	}

	events := make([]Event, len(eventTypes))

	for i := range eventTypes {
		evtVersion := version.Add(uint(i + 1))

		evt, err := New(
			eventTypes[i],
			aggregateID,
			aggregateType,
			evtVersion,
			payloads[i],
			opts...,
		)
		if err != nil {
			return nil, WrapCorruption(
				err,
				"event.create_failed",
				"create event "+strconv.Itoa(i),
			)
		}

		events[i] = evt
	}

	return events, nil
}
