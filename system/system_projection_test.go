package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Projection E2E types ──

type TaskView struct {
	Title  string
	Status string
}

type FindTask struct {
	ID string
}

func projectionDecoder(eventType string, payload []byte) (any, error) {
	switch eventType {
	case "task.created":
		var e TaskCreated
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}

		return e, nil
	}

	return nil, errors.New("unknown event type: " + eventType)
}

func TestSystem_ProjectionE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	taskViewQuery := metaengine.Query[FindTask, TaskView]("task_views",
		metaengine.OnTyped("task.created", TaskCreated{}, func(e TaskCreated) (string, TaskView) {
			return e.Title, TaskView{Title: e.Title, Status: "pending"}
		}),
	)

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("task already exists")
							}

							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "proj-e2e", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:       []any{taskViewQuery},
		ProjectionDecoder: projectionDecoder,
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Dispatch command BEFORE starting projections — the host will replay from journal.
	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	// Start projection processing.
	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	// Wait for projection host to process the event.
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				// Verify projection data via metaengine store.
				result, err := sys.MetaEngine().Execute(FindTask{ID: "proj-e2e"})
				if err != nil {
					t.Fatalf("store.Execute: %v", err)
				}

				view, ok := result.(TaskView)
				if !ok {
					t.Fatalf("expected TaskView, got %T", result)
				}

				if view.Title != "proj-e2e" {
					t.Fatalf("expected title proj-e2e, got %s", view.Title)
				}

				if view.Status != "pending" {
					t.Fatalf("expected status pending, got %s", view.Status)
				}

				return // success
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	for _, s := range sys.ProjectionHost().Status() {
		if s.Errors > 0 {
			t.Fatalf("projection %q has %d errors", s.Name, s.Errors)
		}
	}

	t.Fatal("projection did not process event within timeout")
}

func TestSystem_ProjectionWithSQLite(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	taskViewQuery := metaengine.Query[FindTask, TaskView]("task_views_sqlite",
		metaengine.OnTyped("task.created", TaskCreated{}, func(e TaskCreated) (string, TaskView) {
			return e.Title, TaskView{Title: e.Title, Status: "pending"}
		}),
	)

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "sqlite-proj", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:       []any{taskViewQuery},
		ProjectionDecoder: projectionDecoder,
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: "file:" + t.Name() + "?mode=memory&cache=shared"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				return // success
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	for _, s := range sys.ProjectionHost().Status() {
		if s.Errors > 0 {
			t.Fatalf("projection %q has %d errors", s.Name, s.Errors)
		}
	}

	t.Fatal("projection did not process event within timeout")
}
