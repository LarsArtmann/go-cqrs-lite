package systemscenario

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// journal returns the full event journal, ordered by occurrence.
func (s *Scenario) journal() []event.Event {
	s.t.Helper()

	journal, ok := s.sys.EventStore().(event.Journal)
	if !ok {
		s.t.Fatal("systemscenario: event store does not implement event.Journal (ReadAll)")
	}

	events, err := journal.ReadAll(s.ctx)
	if err != nil {
		s.t.Fatalf("systemscenario: read journal: %v", err)
	}

	return events
}

// journalIDSet returns the set of event IDs currently in the journal.
func (s *Scenario) journalIDSet() map[string]struct{} {
	ids := make(map[string]struct{})

	for _, evt := range s.journal() {
		ids[evt.ID().String()] = struct{}{}
	}

	return ids
}

// commandIDSet returns the set of command IDs captured so far.
func (s *Scenario) commandIDSet() map[string]struct{} {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	ids := make(map[string]struct{}, len(s.capturedCommands))
	for _, entry := range s.capturedCommands {
		ids[entry.cmd.ID().String()] = struct{}{}
	}

	return ids
}

// actEvents returns the journal events emitted since the act baseline,
// preserving journal order.
func (s *Scenario) actEvents() []event.Event {
	events := s.journal()
	acted := make([]event.Event, 0, len(events))

	for _, evt := range events {
		if _, given := s.baselineEventIDs[evt.ID().String()]; !given {
			acted = append(acted, evt)
		}
	}

	return acted
}

// actCommands returns the commands dispatched since the act baseline,
// preserving dispatch order.
func (s *Scenario) actCommands() []command.Command {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	acted := make([]command.Command, 0, len(s.capturedCommands))
	for _, entry := range s.capturedCommands {
		if _, given := s.baselineCommandIDs[entry.cmd.ID().String()]; !given {
			acted = append(acted, entry.cmd)
		}
	}

	return acted
}

// describeEvents renders one line per event for failure diagnostics:
// type, version, stream, and actor.
func describeEvents(events []event.Event) string {
	if len(events) == 0 {
		return "(no events)"
	}

	out := ""

	var outSb315 strings.Builder
	for _, evt := range events {
		fmt.Fprintf(&outSb315, "\n  - %s v%d on %s:%s (actor: %s)",
			evt.Type(), evt.Version(), evt.StreamType(), evt.StreamID(), evt.Metadata().ActorID)
	}

	out += outSb315.String()

	return out
}
