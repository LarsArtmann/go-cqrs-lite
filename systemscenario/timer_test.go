package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// timerDeployment is Memory() — the preset already declares the dedicated
// timers engine.
func timerDeployment() system.DeploymentConfig {
	return systemscenario.Memory()
}

// timerDomain wraps taskDomain with a deadline timer scheduled one hour
// after the scenario's (frozen) clock epoch: when it fires, the scheduler
// dispatches task.complete on the target stream. The scheduler reads the
// SAME clock via scheduling.WithClock(sys.Clock().Now), so TimeAdvances
// fires it deterministically — no sleeping (ADR-0153 D4).
func timerDomain(ref id.StreamRef) system.DomainConfig {
	base := taskDomain()
	baseCommands := base.Commands

	base.Commands = func(sys *system.System) {
		baseCommands(sys)

		store, err := engine.NewTimerStore[string](sys.TimerEngine())
		if err != nil {
			panic(err)
		}

		fireAt := sys.Clock().Now().Add(time.Hour)
		if err := store.Schedule(context.Background(), scheduling.Timer[string]{
			ID:      scheduling.MustParseTimerID("task-deadline"),
			FireAt:  fireAt,
			Payload: ref.ID.String(),
		}); err != nil {
			panic(err)
		}

		sys.ManageTimers(scheduling.New(
			store,
			func(ctx context.Context, timer scheduling.Timer[string]) error {
				return sys.CommandDispatcher().Dispatch(ctx, newTaskCmd("task.complete", ref.ID))
			},
			scheduling.WithPollInterval(5*time.Millisecond),
			scheduling.WithClock(sys.Clock().Now),
		))
	}

	return base
}

// TestSystem_TimeAdvancesFiresDeadlineTimer is the Axon timeElapses analog:
// a frozen clock holds the deadline timer back; advancing past FireAt fires
// it, the scheduler dispatches task.complete, and the awaited Then* sees the
// journal trail plus the captured command chain.
func TestSystem_TimeAdvancesFiresDeadlineTimer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, timerDomain(ref), timerDeployment())

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "deadline", Status: "pending"},
		),
	).TimeAdvances(2 * time.Hour).
		Then("task.updated").
		ThenCommands("task.complete")
}

// TestSystem_TimeAdvancesWithoutTimersIsQuiet pins the edge: advancing with
// no due timers changes nothing — an awaited ThenNoEvents passes after the
// (bounded) await window.
func TestSystem_TimeAdvancesWithoutTimersIsQuiet(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment(),
		systemscenario.WithAwaitTimeout(150*time.Millisecond))

	sc.TimeAdvances(time.Minute).
		ThenNoEvents().
		ThenNoCommands()
}
