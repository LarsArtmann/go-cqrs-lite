package scenario

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// DecideFunc is a pure function that takes the current state and a command,
// and returns the events to emit. It is command-first (it receives the Cmd the
// test is exercising), unlike [decider.DecideFunc] which is version-first
// (it receives the stream version for optimistic concurrency). The two
// intentionally stay separate so this module's dependency footprint stays
// minimal — importing decider would transitively pull snapshot, otel, and
// storage/memory into every scenario consumer.
type DecideFunc[Cmd any, State any] func(state State, cmd Cmd) ([]event.Event, error)

// --- Decider DSL ---

// DeciderScenario is a fluent Given/When/Then builder for testing deciders.
type DeciderScenario[Cmd any, State any] struct {
	apply    func(State, event.Event) (State, error)
	decide   DecideFunc[Cmd, State]
	initial  State
	given    []event.Event
	cmd      Cmd
	t        *testing.T
	asserted bool
}

// Given creates a decider scenario with the given fold function and pre-existing events.
// The fold function is the same as decider.Decider.Apply — it folds an event into state.
func Given[Cmd, State any](
	t *testing.T,
	apply func(State, event.Event) (State, error),
	initial State,
	events ...event.Event,
) *DeciderScenario[Cmd, State] {
	t.Helper()

	s := &DeciderScenario[Cmd, State]{ //nolint:exhaustruct_v5 // cmd+decide set by When()
		t:       t,
		apply:   apply,
		initial: initial,
		given:   events,
	}
	s.requireTerminalAssertion()

	return s
}

// GivenState is a convenience variant of [Given] for the common case where the
// command type parameter is unused — the decide function is called inline in
// When with nil as the command. This eliminates the redundant [any] type
// parameter: GivenState[State](...) instead of Given[any, State](...).
func GivenState[State any](
	t *testing.T,
	apply func(State, event.Event) (State, error),
	initial State,
	events ...event.Event,
) *DeciderScenario[any, State] {
	t.Helper()

	s := &DeciderScenario[any, State]{ //nolint:exhaustruct_v5 // cmd+decide set by When()
		t:       t,
		apply:   apply,
		initial: initial,
		given:   events,
	}
	s.requireTerminalAssertion()

	return s
}

// requireTerminalAssertion registers a cleanup that fails the test if no Then*
// method ever ran — a scenario without a terminal assertion would otherwise
// pass vacuously while asserting nothing.
func (s *DeciderScenario[Cmd, State]) requireTerminalAssertion() {
	s.t.Cleanup(func() {
		if !s.asserted {
			s.t.Errorf(
				"scenario: no Then* assertion ran — this test passes vacuously; " +
					"end the chain with Then, ThenEvents, ThenError, or ThenState",
			)
		}
	})
}

// When sets the command to execute against the folded state.
func (s *DeciderScenario[Cmd, State]) When(
	cmd Cmd,
	decide DecideFunc[Cmd, State],
) *DeciderScenario[Cmd, State] {
	s.cmd = cmd
	s.decide = decide

	return s
}

// requireWhen aborts the test if When was not called first. methodName is the
// name of the Then* variant that the caller belongs to, used in the failure
// message to pinpoint which assertion was misused.
func (s *DeciderScenario[Cmd, State]) requireWhen(methodName string) {
	s.t.Helper()

	if s.decide == nil {
		s.t.Fatal("testing: call When() before " + methodName + "()")
	}

	s.asserted = true
}

// foldGiven folds the Given events into a fresh copy of the initial state.
// Aborts the test on fold error. Used at the start of every Then* method to
// reconstruct the precondition state before invoking decide.
func (s *DeciderScenario[Cmd, State]) foldGiven() State {
	state := s.initial

	for _, evt := range s.given {
		var err error

		state, err = s.apply(state, evt)
		if err != nil {
			s.t.Fatalf("Given: fold failed: %v", err)
		}
	}

	return state
}

// prepareThen marks the helper, asserts When was called, and folds the given events.
func (s *DeciderScenario[Cmd, State]) prepareThen(method string) State {
	s.t.Helper()
	s.requireWhen(method)

	return s.foldGiven()
}

// decideEvents prepares the Then-step state, runs decide, and fails the test
// if decide errors — the shared prologue of Then and ThenEvents.
func (s *DeciderScenario[Cmd, State]) decideEvents(method string) []event.Event {
	state := s.prepareThen(method)

	events, err := s.decide(state, s.cmd)
	if err != nil {
		s.t.Fatalf("%s: decide returned error: %v", method, err)
	}

	return events
}

// Then asserts that the decide function produced the expected event types.
// It compares event types (not payloads) — use ThenFull for deep comparison.
func (s *DeciderScenario[Cmd, State]) Then(expectedEventTypes ...event.Type) {
	events := s.decideEvents("Then")

	gotTypes := make([]event.Type, len(events))
	for i, e := range events {
		gotTypes[i] = e.Type()
	}

	if len(gotTypes) != len(expectedEventTypes) || !slices.Equal(gotTypes, expectedEventTypes) {
		s.t.Fatalf("Then: expected event types %v, got %v", expectedEventTypes, gotTypes)
	}
}

// ThenEvents hands the full events produced by the decide function to inspect
// for arbitrary assertions beyond event types and folded state — metadata
// (actor, correlation), payloads, versions. Use t.Errorf inside inspect so
// remaining assertions still run. Returns the scenario for chaining:
//
//	scenario.Given[Cmd, State](t, apply, initial).
//	    When(cmd, decide).
//	    ThenEvents(func(events []event.Event) {
//	        if events[0].Metadata().ActorID.IsZero() {
//	            t.Error("expected actor on emitted event")
//	        }
//	    }).
//	    Then("user.created")
func (s *DeciderScenario[Cmd, State]) ThenEvents(
	inspect func(events []event.Event),
) *DeciderScenario[Cmd, State] {
	events := s.decideEvents("ThenEvents")

	inspect(events)

	return s
}

// ThenError asserts that the decide function returns an error matching target.
func (s *DeciderScenario[Cmd, State]) ThenError(target error) {
	state := s.prepareThen("ThenError")

	_, err := s.decide(state, s.cmd)
	if err == nil {
		s.t.Fatal("ThenError: expected error, got nil")
	}

	if !errors.Is(err, target) {
		s.t.Fatalf("ThenError: expected %v, got %v", target, err)
	}
}

// ThenState folds the produced events and asserts the resulting state.
func (s *DeciderScenario[Cmd, State]) ThenState(
	apply func(State, event.Event) (State, error),
	initial State,
	expected State,
) {
	s.t.Helper()
	s.requireWhen("ThenState")

	state := initial

	for _, evt := range s.given {
		state = foldOrFatal(s.t, apply, state, evt, "Given: fold failed")
	}

	events, err := s.decide(state, s.cmd)
	if err != nil {
		s.t.Fatalf("When: decide returned error: %v", err)
	}

	for _, evt := range events {
		state = foldOrFatal(s.t, apply, state, evt, "ThenState: fold produced event failed")
	}

	if !reflect.DeepEqual(state, expected) {
		s.t.Fatalf("ThenState: expected %v, got %v", expected, state)
	}
}

// foldOrFatal applies one event and fails the scenario test with failureMsg
// when the fold errors. Shared by the Given and produced-events loops of
// ThenState.
func foldOrFatal[State any](
	tb testing.TB,
	apply func(State, event.Event) (State, error),
	state State,
	evt event.Event,
	failureMsg string,
) State {
	state, err := apply(state, evt)
	if err != nil {
		tb.Fatalf("%s: %v", failureMsg, err)
	}

	return state
}
