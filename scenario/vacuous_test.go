package scenario_test

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

// The vacuous-pass guard fails the host test at cleanup, so "the guard fires"
// can only be observed from outside the failing test process. The env-gated
// child test below deliberately runs scenario chains with no terminal Then*
// and is expected to exit non-zero; the driver test re-execs the test binary
// with the gate set and asserts that exit status plus the guard messages.

const vacuousChildEnv = "SCENARIO_VACUOUS_CHILD"

func TestVacuousGuard_childRunsVacuousScenarios(t *testing.T) {
	if os.Getenv(vacuousChildEnv) != "1" {
		t.Skip("runs only via TestVacuousGuard_FailsWithoutTerminalAssertion")
	}

	scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{}).
		When(incrementCmd{}, decideIncrement)

	scenario.GivenState[counterState](t, foldCounter, counterState{})

	scenario.GivenProjection(t, &testProj{}, mustEvent(evtIncremented))

	scenario.GivenProjection(t, &failingProj{err: errors.New("boom")}, mustEvent(evtIncremented))
}

func TestVacuousGuard_FailsWithoutTerminalAssertion(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(os.Args[0], "-test.run=^TestVacuousGuard_childRunsVacuousScenarios$", "-test.count=1")
	cmd.Env = append(os.Environ(), vacuousChildEnv+"=1")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected the vacuous-scenario child test to FAIL, but the process exited 0")
	}

	output := string(out)
	for _, marker := range []string{
		"no Then* assertion ran",
		"passes vacuously",
		"swallows every handler error",
	} {
		if !strings.Contains(output, marker) {
			t.Errorf("guard output missing %q:\n%s", marker, output)
		}
	}
}

func TestVacuousGuard_DeciderWithThenPasses(t *testing.T) {
	t.Parallel()

	scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{}).
		When(incrementCmd{}, decideIncrement).
		Then(evtIncremented)
}

func TestVacuousGuard_DeciderWithThenStatePasses(t *testing.T) {
	t.Parallel()

	scenario.Given[incrementCmd, counterState](
		t, foldCounter, counterState{},
		mustEvent(evtIncremented),
	).
		When(incrementCmd{}, decideIncrement).
		ThenState(foldCounter, counterState{}, counterState{Count: 2})
}

func TestVacuousGuard_ProjectionWithThenNoErrorPasses(t *testing.T) {
	t.Parallel()

	scenario.GivenProjection(t, &testProj{}, mustEvent(evtIncremented)).
		ThenNoError()
}

func TestVacuousGuard_ProjectionWithThenErrorPasses(t *testing.T) {
	t.Parallel()

	scenario.GivenProjection(t, &failingProj{err: errors.New("boom")}, mustEvent(evtIncremented)).
		ThenError()
}
