//go:build cgo

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
	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func init() {
	system.RegisterDriver(
		"duckdb",
		func(_ context.Context, cfg system.EngineConfig) (metaengine.Engine, error) {
			return duckdbengine.New(cfg.DSN)
		},
	)
}

// TestIntegration_DuckDBSource_HealthCheck verifies a DuckDB-backed
// source-of-truth: dispatch a command, persist an event, load it back,
// run HealthCheck/HealthCheckDetailed, and close cleanly.
func TestIntegration_DuckDBSource_HealthCheck(t *testing.T) {
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
								TaskCreated{Title: "duckdb-integration", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"duckdb-store": {Driver: "duckdb", DSN: ""},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "duckdb-store"},
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

	// Verify the event was persisted.
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

	if p.Title != "duckdb-integration" {
		t.Fatalf("expected title duckdb-integration, got %s", p.Title)
	}

	// HealthCheck should pass — DuckDB implements HealthChecker.
	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// HealthCheckDetailed should include the duckdb engine and report it healthy.
	detailed := sys.HealthCheckDetailed(ctx)
	found := false

	for _, h := range detailed {
		if h.Error != nil {
			t.Fatalf("engine %s unhealthy: %v", h.Name, h.Error)
		}

		if h.Name == "duckdb-store" {
			found = true
		}
	}

	if !found {
		t.Fatalf("duckdb-store not in HealthCheckDetailed: %+v", detailed)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
