package systemscenario

import (
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ThenError asserts the most recent command/query act returned an error
// matching target per errors.Is (Axon then().exception analog).
func (p *WhenPhase) ThenError(target error) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenError")

	if s.lastErr == nil {
		s.t.Fatalf("ThenError: expected error %v, got nil", target)
	}

	if !errors.Is(s.lastErr, target) {
		s.t.Fatalf("ThenError: error mismatch\nwant: %v\ngot:  %v", target, s.lastErr)
	}

	return p
}

// ThenErrorFamily asserts the most recent act's error classifies into the
// given errorfamily family (Rejection, Conflict, Transient, Infrastructure,
// Corruption, Orchestration) — the fleet-standard error assertion.
func (p *WhenPhase) ThenErrorFamily(family errorfamily.Family) *WhenPhase {
	s := p.sc
	s.t.Helper()
	s.requireAct("ThenErrorFamily")

	if s.lastErr == nil {
		s.t.Fatalf("ThenErrorFamily: expected %s error, got nil", family)
	}

	if got := errorfamily.Classify(s.lastErr); got != family {
		s.t.Fatalf("ThenErrorFamily: family mismatch\nwant: %s\ngot:  %s (error: %v)",
			family, got, s.lastErr)
	}

	return p
}
