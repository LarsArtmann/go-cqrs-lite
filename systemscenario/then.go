package systemscenario

import (
	"reflect"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Then asserts the event types emitted by the When acts, in order, by
// diffing the journal against the act baseline (Axon then().events analog).
// Payloads are ignored — use [WhenPhase.ThenEvents] or [ThenPayload].
//
// After [WhenPhase.TimeAdvances] the assertion polls until the types match
// or the await timeout expires (timer firing is asynchronous).
func (p *WhenPhase) Then(expected ...event.Type) {
	s := p.sc
	s.t.Helper()
	s.requireAct("Then")

	if s.awaitMode {
		s.await("Then", func() (bool, string) {
			got := eventTypes(s.actEvents())
			if slices.Equal(got, expected) {
				return true, ""
			}

			return false, "want event types " + formatTypes(expected) +
				" since the When act, got " + formatTypes(got) + describeEvents(s.actEvents())
		})

		return
	}

	got := eventTypes(s.actEvents())
	if !slices.Equal(got, expected) {
		s.t.Fatalf("Then: want event types %s since the When act, got %s\nact events:%s",
			formatTypes(expected), formatTypes(got), describeEvents(s.actEvents()))
	}
}

// ThenEvents hands the full act events to inspect for assertions beyond
// types — payloads, metadata, actor, versions. Use t.Errorf inside inspect
// so remaining assertions still run. Synchronous; for poll-based inspection
// (after TimeAdvances) use [WhenPhase.ThenEventsSatisfy].
func (p *WhenPhase) ThenEvents(inspect func(events []event.Event)) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenEvents")

	inspect(s.actEvents())

	return p
}

// ThenEventsSatisfy polls inspect until it returns nil or the await timeout
// expires. Use after asynchronous acts (TimeAdvances); ThenEvents is the
// synchronous sibling.
func (p *WhenPhase) ThenEventsSatisfy(inspect func(events []event.Event) error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenEventsSatisfy")

	s.await("ThenEventsSatisfy", func() (bool, string) {
		if err := inspect(s.actEvents()); err != nil {
			return false, err.Error()
		}

		return true, ""
	})

	return p
}

// ThenNoEvents asserts the When acts emitted no journal events. After
// TimeAdvances this waits the full await timeout for silence before passing
// — set WithAwaitTimeout to bound it.
func (p *WhenPhase) ThenNoEvents() *WhenPhase {
	p.Then()

	return p
}

// ThenPayload asserts that the i-th act event's payload decodes to want
// (deep-equal, codec auto-detected from the event's encoding stamp). It is a
// package-level generic because Go methods cannot take type parameters.
//
//	systemscenario.ThenPayload(phase, 0, TaskCreated{Title: "ship"})
func ThenPayload[T any](p *WhenPhase, index int, want T) {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenPayload")

	events := s.actEvents()
	if index < 0 || index >= len(events) {
		s.t.Fatalf("ThenPayload: index %d out of range — %d act event(s)%s",
			index, len(events), describeEvents(events))
	}

	got, err := event.DecodePayloadAuto[T](events[index])
	if err != nil {
		s.t.Fatalf("ThenPayload: decode %s: %v", events[index].Type(), err)
	}

	if !equalDeep(got, want) {
		s.t.Fatalf("ThenPayload: %s payload mismatch\nwant: %#v\ngot:  %#v",
			events[index].Type(), want, got)
	}
}

// ThenMetadata spot-checks the i-th act event's metadata (actor, correlation,
// causation, custom data). check returns nil to pass, or an error describing
// the mismatch.
func (p *WhenPhase) ThenMetadata(index int, check func(md event.Metadata) error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenMetadata")

	events := s.actEvents()
	if index < 0 || index >= len(events) {
		s.t.Fatalf("ThenMetadata: index %d out of range — %d act event(s)%s",
			index, len(events), describeEvents(events))
	}

	if err := check(events[index].Metadata()); err != nil {
		s.t.Fatalf("ThenMetadata: %s v%d: %v", events[index].Type(), events[index].Version(), err)
	}

	return p
}

// eventTypes maps events to their types.
func eventTypes(events []event.Event) []event.Type {
	types := make([]event.Type, len(events))
	for i, evt := range events {
		types[i] = evt.Type()
	}

	return types
}

// formatTypes renders a type slice for failure messages.
func formatTypes(types []event.Type) string {
	if len(types) == 0 {
		return "(none)"
	}

	out := "["
	for i, t := range types {
		if i > 0 {
			out += " "
		}

		out += string(t)
	}

	return out + "]"
}

// equalDeep is the deep-equality used by payload assertions.
func equalDeep(got, want any) bool {
	return reflect.DeepEqual(got, want)
}
