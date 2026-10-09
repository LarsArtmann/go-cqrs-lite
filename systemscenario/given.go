package systemscenario

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// Given seeds the scenario with pre-existing events. Events are appended to
// the journal per stream (no concurrency checks) and published to the event
// bus in the same save→publish order the decider repository uses, so
// projections and saga subscribers observe the given history exactly as they
// would in production. Mint correctly-versioned events with [Scenario.Event].
//
// The returned phase supports dispatch-based seeding (Command) and the When
// transition:
//
//	sc.Given(created).When(cmdComplete).Then("task.completed")
//	sc.Given().Command(cmdRegister).When(cmdRename).ThenError(...)
func (s *Scenario) Given(events ...event.Event) *GivenPhase {
	s.t.Helper()

	return (&GivenPhase{sc: s}).Events(events...)
}

// GivenPhase is the seeding phase of a scenario.
type GivenPhase struct {
	sc *Scenario
}

// Events appends more pre-existing events to the given history. See
// [Scenario.Given] for the seeding semantics.
func (g *GivenPhase) Events(events ...event.Event) *GivenPhase {
	s := g.sc
	s.t.Helper()

	if s.actStarted {
		s.t.Fatal("systemscenario: Given must run before the first When act")
	}

	if len(events) == 0 {
		return g
	}

	s.appendEvents(events)
	s.publishEvents(events)

	return g
}

// appendGiven appends the events to the journal, grouped per stream.
func (s *Scenario) appendEvents(events []event.Event) {
	s.t.Helper()

	byStream := make(map[id.StreamRef][]event.Event)
	order := make([]id.StreamRef, 0)

	for _, evt := range events {
		ref := id.NewStreamRef(evt.StreamType(), evt.StreamID())
		if _, seen := byStream[ref]; !seen {
			order = append(order, ref)
		}

		byStream[ref] = append(byStream[ref], evt)
	}

	for _, ref := range order {
		if err := s.sys.EventStore().AppendBatch(s.ctx, ref, byStream[ref]); err != nil {
			s.t.Fatalf("systemscenario.Given: append %s: %v", ref, err)
		}
	}
}

// publishGiven publishes the events to the bus so projections and saga
// subscribers fold them, mirroring the repository's save→publish order.
func (s *Scenario) publishEvents(events []event.Event) {
	s.t.Helper()

	if err := s.sys.Publisher().Publish(s.ctx, events...); err != nil {
		s.t.Fatalf("systemscenario.Given: publish: %v", err)
	}
}

// Command seeds the scenario by intent: the commands dispatch through the
// real dispatcher, so their emitted events land in the journal and flow to
// projections exactly as production traffic (Axon given().command analog).
// Dispatch failures are fatal — givens are preconditions, not acts under
// test. Commands dispatched here are part of the baseline; Then* assertions
// see only the When acts.
func (g *GivenPhase) Command(cmds ...command.Command) *GivenPhase {
	s := g.sc
	s.t.Helper()

	if s.actStarted {
		s.t.Fatal("systemscenario: Given.Command must run before the first When act")
	}

	for _, cmd := range cmds {
		if err := s.sys.CommandDispatcher().Dispatch(s.ctx, cmd); err != nil {
			s.t.Fatalf("systemscenario.Given.Command(%s): %v", cmd.Type(), err)
		}
	}

	return g
}

// When transitions to the act under test by dispatching cmd.
func (g *GivenPhase) When(cmd command.Command) *WhenPhase {
	return g.sc.When(cmd)
}

// WhenEvent transitions to the bus-path act. See [Scenario.WhenEvent].
func (g *GivenPhase) WhenEvent(events ...event.Event) *WhenPhase {
	return g.sc.WhenEvent(events...)
}

// WhenQuery transitions to the act under test by dispatching a query.
func (g *GivenPhase) WhenQuery(q query.Query) *WhenPhase {
	return g.sc.WhenQuery(q)
}

// TimeAdvances transitions to the time act. See [Scenario.TimeAdvances].
func (g *GivenPhase) TimeAdvances(d time.Duration) *WhenPhase {
	return g.sc.TimeAdvances(d)
}
