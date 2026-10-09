package systemscenario

import (
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// ThenCommands asserts the command types dispatched since the act baseline,
// in order — the saga/deriver assertion: an event published on the bus
// triggers a deriver whose derived command is captured here (Axon
// then().commands analog). After TimeAdvances the assertion polls.
func (p *WhenPhase) ThenCommands(expected ...command.Type) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenCommands")

	if s.awaitMode {
		s.await("ThenCommands", func() (bool, string) {
			got := commandTypes(s.actCommands())
			if slices.Equal(got, expected) {
				return true, ""
			}

			return false, "want dispatched command types " + formatTypes(expected) +
				", got " + formatTypes(got) + describeCommands(s.actCommands())
		})

		return p
	}

	got := commandTypes(s.actCommands())
	if !slices.Equal(got, expected) {
		s.t.Fatalf(
			"ThenCommands: want dispatched command types %s since the When act, got %s\nact commands:%s",
			formatTypes(expected),
			formatTypes(got),
			describeCommands(s.actCommands()),
		)
	}

	return p
}

// ThenCommandsSatisfy hands the commands dispatched since the act baseline
// to inspect for assertions beyond types — payloads, stream targets, actors.
// Use t.Errorf inside inspect so remaining assertions still run.
func (p *WhenPhase) ThenCommandsSatisfy(inspect func(cmds []command.Command)) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenCommandsSatisfy")

	inspect(s.actCommands())

	return p
}

// ThenNoCommands asserts no commands were dispatched since the act baseline.
func (p *WhenPhase) ThenNoCommands() *WhenPhase {
	p.ThenCommands()

	return p
}

// commandTypes maps captured commands to their types.
func commandTypes(cmds []command.Command) []command.Type {
	types := make([]command.Type, len(cmds))
	for i, cmd := range cmds {
		types[i] = cmd.Type()
	}

	return types
}

// describeCommands renders one line per command for failure diagnostics.
func describeCommands(cmds []command.Command) string {
	if len(cmds) == 0 {
		return "(no commands)"
	}

	out := ""

	var outSb82 strings.Builder
	for _, cmd := range cmds {
		outSb82.WriteString("\n  - " + string(cmd.Type()) + " on " + cmd.StreamID().String())
	}

	out += outSb82.String()

	return out
}
