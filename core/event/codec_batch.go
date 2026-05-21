package event

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	expectedVersion Version,
	eventTypes []Type,
	payloads []any,
	options ...Option,
) ([]Event, error) {
	if len(eventTypes) != len(payloads) {
		return nil, errMismatchedSlices
	}

	events := make([]Event, 0, len(eventTypes))

	for i, eventType := range eventTypes {
		data, err := json.Marshal(payloads[i])
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errPayloadMarshal, eventType, err)
		}

		evt, err := NewEvent(
			eventType,
			aggregateID,
			aggregateType,
			expectedVersion.Add(i+1),
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

func mustNewEvents(
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	expectedVersion Version,
	eventTypes []Type,
	payloads []any,
	options ...Option,
) []Event {
	events, err := newEvents(
		aggregateID,
		aggregateType,
		expectedVersion,
		eventTypes,
		payloads,
		options...,
	)
	if err != nil {
		panic(fmt.Sprintf("event.mustNewEvents: %v", err))
	}

	return events
}

func decodePayloads[T any](events []Event, codec Codec) ([]T, error) {
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
