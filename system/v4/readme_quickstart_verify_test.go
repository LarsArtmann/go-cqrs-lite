package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type readmeTaskCreated struct {
	Title string
	At    time.Time
}

type readmeTaskState struct {
	Title  string
	Status string
	Exists bool
}

func applyReadmeTask(state readmeTaskState, evt event.Event) (readmeTaskState, error) {
	switch evt.Type() {
	case "task.created":
		var p readmeTaskCreated
		_ = json.Unmarshal(evt.Payload(), &p)
		state.Title = p.Title
		state.Status = "pending"
		state.Exists = true
	}
	return state, nil
}

// TestReadmeQuickStart mirrors system/README.md Quick Start verbatim.
func TestReadmeQuickStart(t *testing.T) {
	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", decider.Decider[readmeTaskState]{
				Initial: readmeTaskState{},
				Apply:   applyReadmeTask,
			})

			system.RegisterCommand[*command.BasicCommand, readmeTaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[readmeTaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state readmeTaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("task already exists")
							}
							evt, err := event.New("task.created", cmd.StreamID(), "Task", ver+1,
								readmeTaskCreated{Title: "my first task", At: time.Now()})
							if err != nil {
								return nil, err
							}
							return []event.Event{evt}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatal(err)
	}
	defer sys.Close()

	if err := sys.Start(ctx); err != nil {
		t.Fatal(err)
	}

	taskID := id.NewStreamID()
	createCmd, err := command.New("task.create", taskID)
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.CommandDispatcher().Dispatch(ctx, createCmd); err != nil {
		t.Fatal(err)
	}

	ref := id.NewStreamRef("Task", taskID)
	events, err := sys.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	t.Logf("created task with %d event(s)", len(events))
}
