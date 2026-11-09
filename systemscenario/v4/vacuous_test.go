package systemscenario_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// The vacuous-pass guard fails the host test at cleanup, so "the guard
// fires" can only be observed from outside the failing test process. The
// env-gated child test deliberately runs scenario chains with no terminal
// Then*; the driver re-execs the test binary with the gate set and asserts
// the exit status plus the guard message. Twin of scenario/vacuous_test.go.

const vacuousChildEnv = "SYSSCEN_VACUOUS_CHILD"

func TestVacuousGuard_childRunsVacuousScenarios(t *testing.T) {
	if os.Getenv(vacuousChildEnv) != "1" {
		t.Skip("runs only via TestVacuousGuard_FailsWithoutTerminalAssertion")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scenario := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())
	ref := id.NewStreamRef("Task", id.NewStreamID())

	scenario.Given(scenario.Event("task.created", ref, TaskCreated{Status: "pending"})).
		When(newTaskCmd("task.complete", ref.ID))
}

func TestVacuousGuard_FailsWithoutTerminalAssertion(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestVacuousGuard_childRunsVacuousScenarios$",
		"-test.count=1",
	)
	cmd.Env = append(os.Environ(), vacuousChildEnv+"=1")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected the vacuous-scenario child test to FAIL, but the process exited 0")
	}

	if !strings.Contains(string(out), "passes vacuously") {
		t.Fatalf("expected the vacuous-guard message in output, got:\n%s", out)
	}
}

// TestFailureDiagnostics asserts the quality of Then failure output: the
// want/got types AND the per-event stream/version/actor context must all be
// present (ADR-0153 D2, F09.5). Same child-process pattern: Fatalf output is
// only observable from outside the failing process.
func TestFailureDiagnostics(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestFailureDiagnostics_childFailsThen$",
		"-test.count=1",
	)
	cmd.Env = append(os.Environ(), vacuousChildEnv+"=1")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected the failing-Then child test to FAIL, but the process exited 0")
	}

	for _, want := range []string{
		"want event types [task.bogus]",
		"got [task.updated]",
		"(actor: ", // per-event actor diagnostics
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("failure output missing %q; got:\n%s", want, out)
		}
	}
}

func TestFailureDiagnostics_childFailsThen(t *testing.T) {
	if os.Getenv(vacuousChildEnv) != "1" {
		t.Skip("runs only via TestFailureDiagnostics")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scenario := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())
	ref := id.NewStreamRef("Task", id.NewStreamID())

	scenario.Given(
		scenario.Event("task.created", ref, TaskCreated{Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.bogus")
}
