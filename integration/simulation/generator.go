package simulation

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// EventGenerator produces deterministic event sequences for stress testing.
type EventGenerator struct {
	streamType id.StreamType
	eventType     event.Type
	payloadGen    func(int) any
}

// NewEventGenerator creates a generator for the given aggregate and event types.
func NewEventGenerator(
	streamType id.StreamType,
	eventType event.Type,
	payloadGen func(int) any,
) *EventGenerator {
	return &EventGenerator{
		streamType: streamType,
		eventType:     eventType,
		payloadGen:    payloadGen,
	}
}

// Generate creates a sequence of events for a single aggregate.
func (g *EventGenerator) Generate(count int) ([]event.Event, error) {
	aggID := id.NewAggregateID()
	types := make([]event.Type, count)
	payloads := make([]any, count)

	for i := range count {
		types[i] = g.eventType
		payloads[i] = g.payloadGen(i)
	}

	events, err := event.NewEvents(aggID, g.streamType, 1, types, payloads)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"simulation.generate",
			"generate %d events",
			count,
		)
	}

	return events, nil
}

// GenerateMulti creates events across multiple aggregates.
func (g *EventGenerator) GenerateMulti(aggregates, eventsPerAggregate int) ([]event.Event, error) {
	var allEvents []event.Event

	for aggIdx := range aggregates {
		aggID := id.NewAggregateID()
		types := make([]event.Type, eventsPerAggregate)
		payloads := make([]any, eventsPerAggregate)

		for i := range eventsPerAggregate {
			types[i] = g.eventType
			payloads[i] = g.payloadGen(aggIdx*eventsPerAggregate + i)
		}

		events, err := event.NewEvents(aggID, g.streamType, 1, types, payloads)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Infrastructure,
				"simulation.generate_multi",
				"generate aggregate %d/%d (eventsPerAggregate=%d)",
				aggIdx+1,
				aggregates,
				eventsPerAggregate,
			)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// DefaultUserGenerator returns a generator that produces UserCreated events.
func DefaultUserGenerator() *EventGenerator {
	return NewEventGenerator(
		"User",
		"UserCreated",
		func(i int) any {
			return map[string]string{
				"name":  fmt.Sprintf("User-%d", i),
				"email": fmt.Sprintf("user%d@example.com", i),
			}
		},
	)
}
