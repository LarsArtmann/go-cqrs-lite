package systemscenario

import (
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Then asserts the event types emitted by the When acts, in order, by
// diffing the journal against the act baseline (Axon then().events analog).
// Payloads are ignored — use [WhenPhase.ThenEvents] or [ThenPayload].
//
// After [WhenPhase.TimeAdvances] the assertion polls until the types match
// or the await timeout expires (timer firing is asynchronous).
func (p *WhenPhase) Then(expected ...event.Type) *WhenPhase {
	s := p.thenScenario("Then")

	if s.awaitMode {
		s.await("Then", func() (bool, string) {
			got := eventTypes(s.actEvents())
			if slices.Equal(got, expected) {
				return true, ""
			}

			return false, "want event types " + formatTypes(expected) +
				" since the When act, got " + formatTypes(got) + describeEvents(s.actEvents())
		})

		return p
	}

	got := eventTypes(s.actEvents())
	if !slices.Equal(got, expected) {
		s.t.Fatalf("Then: want event types %s since the When act, got %s\nact events:%s",
			formatTypes(expected), formatTypes(got), describeEvents(s.actEvents()))
	}

	return p
}

// ThenEvents hands the full act events to inspect for assertions beyond
// types — payloads, metadata, actor, versions. Use t.Errorf inside inspect
// so remaining assertions still run. Synchronous; for poll-based inspection
// (after TimeAdvances) use [WhenPhase.ThenEventsSatisfy].
func (p *WhenPhase) ThenEvents(inspect func(events []event.Event)) *WhenPhase {
	s := p.thenScenario("ThenEvents")

	inspect(s.actEvents())

	return p
}

// ThenEventsSatisfy polls inspect until it returns nil or the await timeout
// expires. Use after asynchronous acts (TimeAdvances); ThenEvents is the
// synchronous sibling.
func (p *WhenPhase) ThenEventsSatisfy(inspect func(events []event.Event) error) *WhenPhase {
	s := p.thenScenario("ThenEventsSatisfy")

	s.await("ThenEventsSatisfy", func() (bool, string) {
		if err := inspect(s.actEvents()); err != nil {
			return false, err.Error()
		}

		return true, ""
	})

	return p
}

// ThenNoEvents asserts the When acts emitted no journal events. Under await
// mode (after TimeAdvances or Await) it watches for silence for the quiet
// window — the full await timeout by default, shorter via
// [WithQuietWindow] — failing as soon as any event appears inside it.
func (p *WhenPhase) ThenNoEvents() *WhenPhase {
	s := p.thenScenario("ThenNoEvents")

	if s.awaitMode {
		window := s.cfg.quietWindow
		if window <= 0 {
			window = s.cfg.awaitTimeout
		}

		deadline := time.Now().Add(window)
		for {
			if events := s.actEvents(); len(events) > 0 {
				s.t.Fatalf("ThenNoEvents: %d event(s) appeared within the %s quiet window:%s",
					len(events), window, describeEvents(events))
			}

			if time.Now().After(deadline) {
				return p
			}

			time.Sleep(10 * time.Millisecond)
		}
	}

	p.Then()

	return p
}

// ThenPayload asserts that the i-th act event's payload decodes to want
// (deep-equal, codec auto-detected from the event's encoding stamp). It is a
// package-level generic because Go methods cannot take type parameters.
//
//	systemscenario.ThenPayload(phase, 0, TaskCreated{Title: "ship"})
func ThenPayload[T any](p *WhenPhase, index int, want T) {
	s := p.thenScenario("ThenPayload")

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
	s := p.thenScenario("ThenMetadata")

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

// thenScenario returns the scenario backing a Then* assertion, marking the
// test-helper boundary and failing fast when no When act ran first — the
// shared prologue of every Then* method.
func (p *WhenPhase) thenScenario(method string) *Scenario {
	s := p.sc
	s.t.Helper()
	s.requireAct(method)

	return s
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
	//art-dupl:accept intentional: per-type diagnostic renderers, not a shared abstraction
	if len(types) == 0 {
		return "(none)"
	}

	out := "["

	var outSb151 strings.Builder

	for i, t := range types {
		if i > 0 {
			outSb151.WriteString(" ")
		}

		outSb151.WriteString(string(t))
	}

	out += outSb151.String()

	return out + "]"
}

// equalDeep is the deep-equality used by payload assertions.
func equalDeep(got, want any) bool {
	return reflect.DeepEqual(got, want)
}
