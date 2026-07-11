package decider_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// rapidCounterState is a simple state for property testing.
type rapidCounterState struct {
	Value int
}

func foldRapidCounter(state rapidCounterState, evt event.Event) (rapidCounterState, error) {
	switch evt.Type() {
	case "CounterCreated":
		return rapidCounterState{Value: 1}, nil
	case "CounterIncremented":
		return rapidCounterState{Value: state.Value + 1}, nil
	case "CounterDecremented":
		return rapidCounterState{Value: state.Value - 1}, nil
	}

	return state, nil
}

// generateEvent creates a rapid generator for events.
var generateEvent = rapid.Custom(func(t *rapid.T) event.Event {
	types := []event.Type{"CounterCreated", "CounterIncremented", "CounterDecremented"}
	typ := rapid.SampledFrom(types).Draw(t, "type")
	aggID := id.NewAggregateID()
	version := event.Version(rapid.IntRange(1, 100).Draw(t, "version"))

	evt, err := event.NewEvent(typ, aggID, "Counter", version, nil)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
})

// TestDeterministicApply checks that applying the same sequence of events
// always produces the same state.
func TestDeterministicApply(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		events := rapid.SliceOfN(generateEvent, 0, 50).Draw(t, "events")

		d := decider.Decider[rapidCounterState]{
			Initial: rapidCounterState{},
			Apply:   foldRapidCounter,
		}

		state1, err1 := foldAll(d.Initial, d.Apply, events)
		state2, err2 := foldAll(d.Initial, d.Apply, events)

		if err1 != nil || err2 != nil {
			t.Skip("apply error")
		}

		if state1 != state2 {
			t.Fatalf("apply is not deterministic: %+v vs %+v", state1, state2)
		}
	})
}

// TestVersionMonotonicity checks that event versions are strictly increasing
// within a batch created by NewEvents.
func TestVersionMonotonicity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 20).Draw(t, "count")
		aggID := id.NewAggregateID()
		startVersion := event.Version(rapid.IntRange(1, 100).Draw(t, "startVersion"))

		types := make([]event.Type, count)
		payloads := make([]any, count)
		for i := range types {
			types[i] = "CounterIncremented"
			payloads[i] = struct{ Delta int }{Delta: 1}
		}

		events, err := event.NewEvents(aggID, "Counter", startVersion, types, payloads)
		if err != nil {
			t.Fatalf("create events: %v", err)
		}

		for i := 1; i < len(events); i++ {
			if events[i].Version() <= events[i-1].Version() {
				t.Fatalf(
					"version not monotonic: %d -> %d",
					events[i-1].Version(),
					events[i].Version(),
				)
			}
		}
	})
}

// TestApplyAccumulation checks that applying produces deterministic results
// and that the apply function correctly applies events.
func TestApplyAccumulation(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		events := rapid.SliceOfN(generateEvent, 1, 30).Draw(t, "events")

		d := decider.Decider[rapidCounterState]{
			Initial: rapidCounterState{},
			Apply:   foldRapidCounter,
		}

		state, err := foldAll(d.Initial, d.Apply, events)
		if err != nil {
			t.Skip("apply error")
		}

		// Manually compute expected state
		expected := rapidCounterState{}
		for _, evt := range events {
			switch evt.Type() {
			case "CounterCreated":
				expected = rapidCounterState{Value: 1}
			case "CounterIncremented":
				expected.Value++
			case "CounterDecremented":
				expected.Value--
			}
		}

		if state != expected {
			t.Fatalf("expected %+v, got %+v", expected, state)
		}
	})
}

func foldAll[State any](
	initial State,
	apply func(State, event.Event) (State, error),
	events []event.Event,
) (State, error) {
	state := initial
	for _, evt := range events {
		var err error
		state, err = apply(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}
