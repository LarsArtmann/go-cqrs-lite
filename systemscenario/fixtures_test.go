package systemscenario_test

// Task-domain fixture shared by the harness self-tests. Twin of the
// systemtest/fixtures_test.go block — test fixtures may not cross the module
// boundary.

import (
	"context"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Domain types ──

type TaskCreated struct {
	ID     string
	Title  string
	Status string
}

type TaskUpdated struct {
	ID     string
	Title  string
	Status string
}

type TaskState struct {
	Title  string
	Status string
	Exists bool
}

type TaskView struct {
	ID     string
	Title  string
	Status string
}

var (
	errTaskExists    error = errorfamily.NewRejection("task.already_exists", "task already exists")
	errTaskMissing   error = errorfamily.NewRejection("task.missing", "task does not exist")
	errTaskCompleted error = errorfamily.NewConflict("task.already_completed", "task already completed")

	// completerActor is stamped on completion task.updated events by the fixture
	// handler so metadata assertions have something deterministic to check.
	completerActor = id.NewActorID(id.ActorSystem, "harness-fixture")
)

// EchoQuery is a deterministic query handler target (no projection
// dependency) for WhenQuery-act tests.
type EchoQuery struct {
	Value string
}

func (EchoQuery) Type() query.Type { return "task.echo" }

func applyTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case "task.created":
		payload, err := event.DecodePayloadAuto[TaskCreated](evt)
		if err != nil {
			return state, err
		}

		state.Title, state.Status, state.Exists = payload.Title, payload.Status, true
	case "task.updated":
		payload, err := event.DecodePayloadAuto[TaskUpdated](evt)
		if err != nil {
			return state, err
		}

		state.Title, state.Status = payload.Title, payload.Status
	}

	return state, nil
}

var taskDecider = decider.Decider[TaskState]{
	Initial: TaskState{},
	Apply:   applyTask,
}

// taskEvent mints a Task-stream event, panicking on construction errors
// (fixture invariant).
func taskEvent(
	eventType event.Type,
	streamID id.StreamID,
	version event.Version,
	payload any,
	opts ...event.Option,
) event.Event {
	evt, err := event.New(eventType, streamID, "Task", version, payload, opts...)
	if err != nil {
		panic(err)
	}

	return evt
}

// newTaskCmd mints a BasicCommand, panicking on construction errors.
func newTaskCmd(cmdType command.Type, streamID id.StreamID) *command.BasicCommand {
	cmd, err := command.New(cmdType, streamID)
	if err != nil {
		panic(err)
	}

	return cmd
}

// taskDomain is the ONE DomainConfig both the harness tests and the
// config-drift demo boot — the fixture-from-production-config property.
func taskDomain() system.DomainConfig {
	return system.DomainConfig{
		Commands: func(sys *system.System) {
			registerTaskHandlers(sys)
		},
		Queries: func(sys *system.System) {
			system.RegisterQuery[EchoQuery, string](sys, "task.echo", //nolint:errcheck // fixture
				func(ctx context.Context, q EchoQuery) (string, error) {
					return q.Value, nil
				})
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[TaskView]("task_views").
				On("task.created", TaskCreated{}).
				On("task.updated", TaskUpdated{}).
				Done(),
		},
	}
}

func registerTaskHandlers(sys *system.System) {
	system.RegisterDecider(sys, "Task", taskDecider) //nolint:errcheck // fixture: registration cannot fail

	system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create", //nolint:errcheck // fixture
		func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), "Task",
				func(state TaskState, version event.Version) ([]event.Event, error) {
					if state.Exists {
						return nil, errTaskExists
					}

					return []event.Event{taskEvent("task.created", cmd.StreamID(), version+1,
						TaskCreated{ID: cmd.StreamID().String(), Title: "first", Status: "pending"})}, nil
				})
		})

	system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.rename", //nolint:errcheck // fixture
		func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), "Task",
				func(state TaskState, version event.Version) ([]event.Event, error) {
					if !state.Exists {
						return nil, errTaskMissing
					}

					return []event.Event{taskEvent("task.updated", cmd.StreamID(), version+1,
						TaskUpdated{ID: cmd.StreamID().String(), Title: "renamed", Status: state.Status})}, nil
				})
		})

	system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.complete", //nolint:errcheck // fixture
		func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), "Task",
				func(state TaskState, version event.Version) ([]event.Event, error) {
					if !state.Exists {
						return nil, errTaskMissing
					}

					if state.Status == "completed" {
						return nil, errTaskCompleted
					}

					return []event.Event{taskEvent("task.updated", cmd.StreamID(), version+1,
						TaskUpdated{ID: cmd.StreamID().String(), Title: state.Title, Status: "completed"},
						event.WithActor(completerActor))}, nil
				})
		})
}

// memoryDeployment mirrors the auto-projection test deployment: one memory
// engine serving source-of-truth and projections.
func memoryDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

// sagaDomain extends taskDomain with the archiver saga: a deriver reacts to
// task.updated events by dispatching task.archive, whose handler emits
// task.archived — the event→command→event chain ThenCommands asserts.
func sagaDomain() system.DomainConfig {
	base := taskDomain()
	baseCommands := base.Commands

	base.Commands = func(sys *system.System) {
		baseCommands(sys)
		registerArchive(sys)

		archiver := deriver.Deriver(func(ctx context.Context, evt event.Event) ([]command.Command, error) {
			return []command.Command{newTaskCmd("task.archive", evt.StreamID())}, nil
		})
		if err := sys.Bus().Subscribe("task.updated", archiver.AsHandler(sys.CommandDispatcher())); err != nil {
			panic(err)
		}
	}

	return base
}

func registerArchive(sys *system.System) {
	system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.archive", //nolint:errcheck // fixture
		func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), "Task",
				func(state TaskState, version event.Version) ([]event.Event, error) {
					if !state.Exists {
						return nil, errTaskMissing
					}

					return []event.Event{taskEvent("task.archived", cmd.StreamID(), version+1,
						TaskUpdated{ID: cmd.StreamID().String(), Title: state.Title, Status: "archived"})}, nil
				})
		})
}
