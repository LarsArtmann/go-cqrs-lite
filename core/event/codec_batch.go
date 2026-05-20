package event

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// NewEvents creates multiple events from typed payloads, auto-marshaling to JSON.
// Events are numbered sequentially starting from expectedVersion+1.
func NewEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	expectedVersion Version,
	eventTypes []Type,
	payloads []any,
	options ...Option,
) ([]Event, error) {
	if len(eventTypes) != len(payloads) {
		return nil, ErrMismatchedSlices
	}

	events := make([]Event, 0, len(eventTypes))

	for i, eventType := range eventTypes {
		data, err := json.Marshal(payloads[i])
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrPayloadMarshal, eventType, err)
		}

		evt, err := NewEvent(
			eventType,
			aggregateID,
			aggregateType,
			Version(expectedVersion.Int()+i+1),
			data,
			options...,
		)
		if err != nil {
			return nil, fmt.Errorf("create event %s: %w", eventType, err)
		}

		events = append(events, evt)
	}

	return events, nil
}

// MustNewEvents creates multiple events or panics.
// Use only in tests where inputs are guaranteed valid.
func MustNewEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	expectedVersion Version,
	eventTypes []Type,
	payloads []any,
	options ...Option,
) []Event {
	events, err := NewEvents(
		aggregateID,
		aggregateType,
		expectedVersion,
		eventTypes,
		payloads,
		options...,
	)
	if err != nil {
		panic(fmt.Sprintf("event.MustNewEvents: %v", err))
	}

	return events
}

// DecodePayloads decodes multiple events' payloads into typed values.
// Returns the decoded values in the same order as the input events.
func DecodePayloads[T any](events []Event, codec Codec) ([]T, error) {
	results := make([]T, 0, len(events))

	for _, evt := range events {
		val, err := DecodePayload[T](evt, codec)
		if err != nil {
			return nil, err
		}

		results = append(results, val)
	}

	return results, nil
}
