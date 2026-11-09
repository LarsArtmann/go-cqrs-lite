package systemscenario

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// When is the act under test: dispatch cmd through the system's command
// dispatcher. The dispatch is synchronous through the journal — event
// assertions after it need no polling. The returned phase accepts further
// chained acts (Event, Query, TimeAdvances) and every Then* assertion.
func (s *Scenario) When(cmd command.Command) *WhenPhase {
	s.t.Helper()

	s.markActStart()

	s.lastErr = s.sys.CommandDispatcher().Dispatch(s.ctx, cmd)

	return &WhenPhase{sc: s}
}

// WhenEvent is the external-event act: events are recorded in the journal
// (facts that happened, ADR-0136) AND published to the event bus, so both
// journal-tailing projections and bus subscribers (derivers, notifications)
// react exactly as they would to production external traffic (Axon
// when().event analog — the aggregate fixture likewise injects into the
// event stream).
func (s *Scenario) WhenEvent(events ...event.Event) *WhenPhase {
	return (&WhenPhase{sc: s}).Event(events...)
}

// TimeAdvances is the time act: advance the scenario's manual clock by d
// (Axon 4 whenTimeElapses analog; Axon 5 dropped it). Timers whose FireAt
// falls due fire on the scheduler's next poll tick, and the scenario flips
// into await mode so subsequent Then* assertions poll. Requires the default
// harness clock (do not pass [WithClock]) and consumer timers wired with
// scheduling.WithClock(sys.Clock().Now).
func (s *Scenario) TimeAdvances(d time.Duration) *WhenPhase {
	return (&WhenPhase{sc: s}).TimeAdvances(d)
}

// WhenQuery is the query act: dispatch q through the query dispatcher and
// capture the result for [WhenPhase.ThenResult] / [WhenPhase.ThenSuccess].
// go-cqrs-lite queries are first-class, so the harness makes them a phase
// (Axon has no whenQuery equivalent).
func (s *Scenario) WhenQuery(q query.Query) *WhenPhase {
	return (&WhenPhase{sc: s}).Query(q)
}

// WhenPhase is the act phase of a scenario: it accepts further chained acts
// and every Then* assertion.
type WhenPhase struct {
	sc *Scenario
}

// beginAct returns the scenario with the helper boundary marked and a fresh
// act baseline started — the shared prologue of every chained act.
func (p *WhenPhase) beginAct() *Scenario {
	s := p.sc
	s.t.Helper()
	s.markActStart()

	return s
}

// Command chains another command act (Axon when().command analog).
func (p *WhenPhase) Command(cmd command.Command) *WhenPhase {
	return p.sc.When(cmd)
}

// Event chains an external-event act: journal + publish. See
// [Scenario.WhenEvent].
func (p *WhenPhase) Event(events ...event.Event) *WhenPhase {
	s := p.beginAct()

	if len(events) > 0 {
		s.journalThenPublish(events)
	}

	return p
}

// Query chains a query act. See [Scenario.WhenQuery].
func (p *WhenPhase) Query(q query.Query) *WhenPhase {
	s := p.beginAct()

	result, err := s.sys.QueryDispatcher().Dispatch(s.ctx, q)
	s.lastQueryResult, s.lastErr = result, err

	return p
}

// TimeAdvances chains the time act. See [Scenario.TimeAdvances].
func (p *WhenPhase) TimeAdvances(d time.Duration) *WhenPhase {
	s := p.beginAct()

	manual, ok := s.clock.(*system.ManualClock)
	if !ok {
		s.t.Fatal("systemscenario: TimeAdvances requires the default harness ManualClock " +
			"(a WithClock option replaced it)")
	}

	manual.Advance(d)

	s.awaitMode = true

	return p
}

// Await flips the scenario into await mode: every subsequent Then*
// assertion polls until it passes or the await timeout expires. The bus
// delivers to subscribers (derivers, notifications) asynchronously, so acts
// whose outcomes arrive via a bus handler — a deriver dispatching a derived
// command — must be awaited:
//
//	sc.When(cmdComplete).Await().
//		Then("task.updated", "task.archived").
//		ThenCommands("task.complete", "task.archive")
//
// [WhenPhase.TimeAdvances] implies await mode.
func (p *WhenPhase) Await() *WhenPhase {
	p.sc.awaitMode = true

	return p
}
