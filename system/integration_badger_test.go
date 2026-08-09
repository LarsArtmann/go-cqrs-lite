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
	badgerengine "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func init() {
	system.RegisterDriver(
		"badger",
		func(_ context.Context, cfg system.EngineConfig) (metaengine.Engine, error) {
			return badgerengine.NewBadgerEngine(
				cfg.DSN,
			) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}

// TestIntegration_BadgerSource_HealthCheck verifies a Badger-backed
// source-of-truth: dispatch a command, persist an event, load it back,
// run HealthCheck/HealthCheckDetailed, and close cleanly.
func TestIntegration_BadgerSource_HealthCheck(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
								TaskCreated{Title: "badger-integration", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"badger-store": {Driver: "badger", DSN: ""},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "badger-store"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	streamID := id.NewStreamID()

	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", streamID)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	ref := id.NewStreamRef("Task", streamID)
	events, err := sys.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	var p TaskCreated
	if err := json.Unmarshal(events[0].Payload(), &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if p.Title != "badger-integration" {
		t.Fatalf("expected title badger-integration, got %s", p.Title)
	}

	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	detailed := sys.HealthCheckDetailed(ctx)
	found := false

	for _, h := range detailed {
		if h.Error != nil {
			t.Fatalf("engine %s unhealthy: %v", h.Name, h.Error)
		}

		if h.Name == "badger-store" {
			found = true
		}
	}

	if !found {
		t.Fatalf("badger-store not in HealthCheckDetailed: %+v", detailed)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
