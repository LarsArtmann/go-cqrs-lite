package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// TestSystem_AdoptTimeAdvancesFiresDeadlineTimer pins the Adopt half of the
// time act: a caller-booted system built with a ManualClock must get a
// working TimeAdvances. Adopt used to leave the scenario clock unset, so
// TimeAdvances failed claiming a WithClock option had replaced the clock.
func TestSystem_AdoptTimeAdvancesFiresDeadlineTimer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref := id.NewStreamRef("Task", id.NewStreamID())

	manual := system.NewManualClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	sys, err := system.New(ctx, timerDomain(ref), timerDeployment(), system.WithClock(manual))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()
		t.Fatalf("Start: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	sc := systemscenario.Adopt(t, ctx, sys)

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "adopted", Status: "pending"},
		),
	).TimeAdvances(2 * time.Hour).
		Then("task.updated").
		ThenCommands("task.complete")
}
