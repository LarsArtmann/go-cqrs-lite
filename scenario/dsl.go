package scenario

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
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
	apply   func(State, event.Event) (State, error)
	decide  DecideFunc[Cmd, State]
	initial State
	given   []event.Event
	cmd     Cmd
	t       *testing.T
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

	return &DeciderScenario[Cmd, State]{ //nolint:exhaustruct // cmd+decide set by When()
		t:       t,
		apply:   apply,
		initial: initial,
		given:   events,
	}
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

	return &DeciderScenario[any, State]{ //nolint:exhaustruct // cmd+decide set by When()
		t:       t,
		apply:   apply,
		initial: initial,
		given:   events,
	}
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

// Then asserts that the decide function produced the expected event types.
// It compares event types (not payloads) — use ThenFull for deep comparison.
func (s *DeciderScenario[Cmd, State]) Then(expectedEventTypes ...event.Type) {
	s.t.Helper()
	s.requireWhen("Then")

	state := s.foldGiven()

	events, err := s.decide(state, s.cmd)
	if err != nil {
		s.t.Fatalf("When: decide returned error: %v", err)
	}

	gotTypes := make([]event.Type, len(events))
	for i, e := range events {
		gotTypes[i] = e.Type()
	}

	if !reflect.DeepEqual(gotTypes, expectedEventTypes) {
		s.t.Fatalf("Then: expected event types %v, got %v", expectedEventTypes, gotTypes)
	}
}

// ThenError asserts that the decide function returns an error matching target.
func (s *DeciderScenario[Cmd, State]) ThenError(target error) {
	s.t.Helper()
	s.requireWhen("ThenError")

	state := s.foldGiven()

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
		var err error

		state, err = apply(state, evt)
		if err != nil {
			s.t.Fatalf("Given: fold failed: %v", err)
		}
	}

	events, err := s.decide(state, s.cmd)
	if err != nil {
		s.t.Fatalf("When: decide returned error: %v", err)
	}

	for _, evt := range events {
		var err error

		state, err = apply(state, evt)
		if err != nil {
			s.t.Fatalf("ThenState: fold produced event failed: %v", err)
		}
	}

	if !reflect.DeepEqual(state, expected) {
		s.t.Fatalf("ThenState: expected %v, got %v", expected, state)
	}
}

// --- Projection DSL ---

// ProjectionScenario tests that a projection handles events without error.
type ProjectionScenario struct {
	proj projection.Projection
	t    *testing.T
	errs []error
}

// GivenProjection creates a projection scenario and feeds it the given events.
func GivenProjection(
	t *testing.T,
	proj projection.Projection,
	events ...event.Event,
) *ProjectionScenario {
	t.Helper()
	scenario := &ProjectionScenario{proj: proj, t: t} //nolint:exhaustruct // errs populated below

	ctx := context.Background()
	for _, evt := range events {
		if err := proj.Handle(ctx, evt); err != nil {
			scenario.errs = append(scenario.errs, errorfamily.Wrap(err, errorfamily.Classify(err),
				"scenario.projection.handle",
				fmt.Sprintf("event %s", evt.Type())))
		}
	}

	return scenario
}

// ThenNoError asserts the projection handled all events without error.
func (s *ProjectionScenario) ThenNoError() {
	s.t.Helper()

	if len(s.errs) > 0 {
		for _, err := range s.errs {
			s.t.Errorf("projection %q: %v", s.proj.Name(), err)
		}

		s.t.FailNow()
	}
}

// ThenError asserts the projection returned at least one error.
func (s *ProjectionScenario) ThenError() {
	s.t.Helper()

	if len(s.errs) == 0 {
		s.t.Fatalf("projection %q: expected at least one error, got none", s.proj.Name())
	}
}
