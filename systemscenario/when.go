package systemscenario

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
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

// Command chains another command act (Axon when().command analog).
func (p *WhenPhase) Command(cmd command.Command) *WhenPhase {
	return p.sc.When(cmd)
}

// Event chains an external-event act: journal + publish. See
// [Scenario.WhenEvent].
func (p *WhenPhase) Event(events ...event.Event) *WhenPhase {
	s := p.sc
	s.t.Helper()

	s.markActStart()

	if len(events) > 0 {
		s.appendEvents(events)
		s.publishEvents(events)
	}

	return p
}

// Query chains a query act. See [Scenario.WhenQuery].
func (p *WhenPhase) Query(q query.Query) *WhenPhase {
	s := p.sc
	s.t.Helper()

	s.markActStart()

	result, err := s.sys.QueryDispatcher().Dispatch(s.ctx, q)
	s.lastQueryResult, s.lastErr = result, err

	return p
}
