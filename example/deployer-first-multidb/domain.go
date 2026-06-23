package main

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type DecideFunc = decider.DecideFunc[CounterState]

const (
	eventCounterIncremented = event.Type("counter.incremented")
	aggregateType           = event.AggregateType("Counter")
)

type CounterIncrementedPayload struct {
	By int `json:"by"`
}

type CounterState struct {
	Count int
}

func applyCounter(state CounterState, evt event.Event) (CounterState, error) {
	if evt.Type() == eventCounterIncremented {
		return CounterState{Count: state.Count + 1}, nil
	}

	return state, nil
}

func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return b, nil
}

func decideIncrement(aggID id.AggregateID) DecideFunc {
	return func(state CounterState, version event.Version) ([]event.Event, error) {
		payload, err := marshalPayload(CounterIncrementedPayload{By: 1})
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "counter.increment.payload", "%v", err)
		}

		evt, err := event.NewEvent(
			eventCounterIncremented, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"counter.increment.1",
				"build event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}
