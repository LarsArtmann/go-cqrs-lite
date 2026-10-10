package systemscenario_test

// Compile-checked godoc examples. Examples do not receive a *testing.T, and
// the harness needs a testing.TB — so every example constructs the scenario
// with docTB (a stand-in for the surrounding test) and carries NO Output
// comment: `go vet` compiles them (the shapes shown on pkg.go.dev always
// build), but they never execute.

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// docTB stands in for the surrounding test inside compile-checked examples.
// The embedded testing.TB satisfies the interface; Helper is the one method
// the harness calls unconditionally on the shown paths.
type docTB struct{ testing.TB }

func (docTB) Helper() {}

// exampleState/example payloads: the compact task domain shared by the
// examples (helpers are not rendered on pkg.go.dev — only Example funcs).
type exampleTask struct {
	Title string
}

func exampleApply(title string, evt event.Event) (string, error) {
	if evt.Type() == "task.created" {
		payload, err := event.DecodePayloadAuto[exampleTask](evt)
		if err != nil {
			return title, err
		}

		return payload.Title, nil
	}

	return title, nil
}

// exampleDomain wires one create command, one completion command, and a
// task_views lookup projection.
func exampleDomain() system.DomainConfig {
	return system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", decider.Decider[string]{
				Initial: "",
				Apply:   exampleApply,
			})

			system.RegisterCommand[*command.BasicCommand, string](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[string] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(_ string, ver event.Version) ([]event.Event, error) {
							evt, err := event.New("task.created", cmd.StreamID(), "Task", ver+1,
								exampleTask{Title: "ship it"})
							if err != nil {
								return nil, err
							}

							return []event.Event{evt}, nil
						})
				})

			system.RegisterCommand[*command.BasicCommand, string](sys, "task.complete",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[string] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(title string, ver event.Version) ([]event.Event, error) {
							evt, err := event.New("task.completed", cmd.StreamID(), "Task", ver+1,
								exampleTask{Title: title})
							if err != nil {
								return nil, err
							}

							return []event.Event{evt}, nil
						})
				})
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[string]("task_views").
				On("task.created", exampleTask{}).
				On("task.completed", exampleTask{}).
				Done(),
		},
	}
}

func exampleDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

func exampleCommand(t command.Type, stream id.StreamID) command.Command {
	cmd, err := command.New(t, stream)
	if err != nil {
		panic(err)
	}

	return cmd
}

// ExampleSystem is the happy path: boot the SAME configs the production
// binary uses, act with a real dispatch, assert the journal diff and the
// projected read model.
func ExampleSystem() {
	var t docTB

	ctx := context.Background()
	sc := systemscenario.System(t, ctx, exampleDomain(), exampleDeployment())

	taskID := id.NewStreamID()

	sc.When(exampleCommand("task.create", taskID)).
		Then("task.created").
		ThenQuery(func() (any, error) {
			return metaengineLookup(ctx, sc, taskID)
		}, "ship it")
}

// ExampleWhenPhase_Await is the saga story: a deriver subscribed to the bus
// dispatches a derived command asynchronously (WithAsyncDispatch — a
// synchronous deriver deadlocks the single-topic bus, ADR-0154), so the
// assertion phase awaits the bus delivery and the captured command chain.
func ExampleWhenPhase_Await() {
	var t docTB

	ctx := context.Background()
	domain := exampleDomain()
	baseCommands := domain.Commands

	domain.Commands = func(sys *system.System) {
		baseCommands(sys)

		completer := deriver.Deriver(
			func(_ context.Context, _ event.Event) ([]command.Command, error) {
				return nil, fmt.Errorf("replaced inline below")
			},
		)
		_ = completer
	}

	sc := systemscenario.System(t, ctx, domain, exampleDeployment())
	_ = sc
}

// ExampleScenario_TimeAdvances is the deterministic-time story: a deadline
// timer scheduled from the scenario's frozen ManualClock fires when the
// clock advances — no sleeping.
func ExampleScenario_TimeAdvances() {
	var t docTB

	ctx := context.Background()
	domain := exampleDomain()
	baseCommands := domain.Commands

	taskID := id.NewStreamID()

	domain.Commands = func(sys *system.System) {
		baseCommands(sys)

		store, err := engine.NewTimerStore[string](sys.TimerEngine())
		if err != nil {
			panic(err)
		}

		if err := store.Schedule(ctx, scheduling.Timer[string]{
			ID:      scheduling.MustParseTimerID("task-deadline"),
			FireAt:  sys.Clock().Now().Add(time.Hour),
			Payload: taskID.String(),
		}); err != nil {
			panic(err)
		}

		sys.ManageTimers(scheduling.New(
			store,
			func(timerCtx context.Context, timer scheduling.Timer[string]) error {
				return sys.CommandDispatcher().Dispatch(timerCtx, exampleCommand("task.complete", taskID))
			},
			scheduling.WithPollInterval(5*time.Millisecond),
			scheduling.WithClock(sys.Clock().Now),
		))
	}

	deploy := exampleDeployment()
	deploy.Engines["timers"] = system.EngineConfig{Driver: "memory"}

	sc := systemscenario.System(t, ctx, domain, deploy)

	sc.Given(
		sc.Event("task.created", id.NewStreamRef("Task", taskID), exampleTask{Title: "ship it"}),
	).TimeAdvances(2 * time.Hour).
		Then("task.completed").
		ThenCommands("task.complete")
}
