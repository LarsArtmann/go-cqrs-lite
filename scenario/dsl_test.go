package scenario_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// --- Test fixtures ---

type (
	counterState struct{ Count int }
	incrementCmd struct{}
	decrementCmd struct{}
)

var (
	evtIncremented = event.Type("CounterIncremented")
	evtDecremented = event.Type("CounterDecremented")
	errEvtLimit    = errors.New("count cannot go below zero")
)

func foldCounter(s counterState, evt event.Event) (counterState, error) {
	switch evt.Type() {
	case evtIncremented:
		s.Count++
	case evtDecremented:
		s.Count--
	}

	return s, nil
}

func decideIncrement(s counterState, _ incrementCmd) ([]event.Event, error) {
	return []event.Event{mustEvent(evtIncremented)}, nil
}

func decideDecrement(s counterState, _ decrementCmd) ([]event.Event, error) {
	if s.Count <= 0 {
		return nil, errEvtLimit
	}

	return []event.Event{mustEvent(evtDecremented)}, nil
}

func mustEvent(t event.Type) event.Event {
	aggID := id.NewAggregateID()
	evt, err := event.New(t, aggID, "Counter", 1, map[string]any{"v": 1})
	if err != nil {
		panic(err)
	}

	return evt
}

// --- Decider tests ---

func TestGiven_When_Then_EventTypes(t *testing.T) {
	t.Parallel()
	scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{}).
		When(incrementCmd{}, decideIncrement).
		Then(evtIncremented)
}

func TestGiven_When_Then_FoldsPriorEvents(t *testing.T) {
	t.Parallel()
	scenario.Given[incrementCmd, counterState](
		t, foldCounter, counterState{},
		mustEvent(evtIncremented),
		mustEvent(evtIncremented),
	).
		When(incrementCmd{}, decideIncrement).
		Then(evtIncremented)
}

func TestGiven_When_ThenError(t *testing.T) {
	t.Parallel()
	scenario.Given[decrementCmd, counterState](t, foldCounter, counterState{}).
		When(decrementCmd{}, decideDecrement).
		ThenError(errEvtLimit)
}

func TestGiven_When_ThenState(t *testing.T) {
	t.Parallel()
	scenario.Given[incrementCmd, counterState](
		t, foldCounter, counterState{},
		mustEvent(evtIncremented),
		mustEvent(evtIncremented),
	).
		When(incrementCmd{}, decideIncrement).
		ThenState(foldCounter, counterState{}, counterState{Count: 3})
}

// --- Projection tests ---

type testProj struct{}

func (p *testProj) Name() string                                  { return "test" }
func (p *testProj) Handle(_ context.Context, _ event.Event) error { return nil }
func (p *testProj) EventTypes() []event.Type                      { return nil }

// failingProj always returns an error on Handle.
type failingProj struct{ err error }

func (p *failingProj) Name() string                                  { return "failing" }
func (p *failingProj) Handle(_ context.Context, _ event.Event) error { return p.err }
func (p *failingProj) EventTypes() []event.Type                      { return nil }

func TestGivenProjection_ThenNoError(t *testing.T) {
	t.Parallel()
	proj := &testProj{}
	scenario.GivenProjection(t, proj, mustEvent(evtIncremented)).
		ThenNoError()
}

func TestGivenProjection_MultipleEvents_ThenNoError(t *testing.T) {
	t.Parallel()
	proj := &testProj{}
	scenario.GivenProjection(
		t, proj,
		mustEvent(evtIncremented),
		mustEvent(evtIncremented),
		mustEvent(evtDecremented),
	).ThenNoError()
}

func TestGivenProjection_ThenError(t *testing.T) {
	t.Parallel()
	proj := &failingProj{err: errors.New("handler exploded")}
	scenario.GivenProjection(t, proj, mustEvent(evtIncremented)).
		ThenError()
}

func TestGivenProjection_NoEvents_ThenNoError(t *testing.T) {
	t.Parallel()
	proj := &testProj{}
	scenario.GivenProjection(t, proj).ThenNoError()
}
