package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4" // self-registers "postgres"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// postgresTestDSN returns a Postgres DSN from POSTGRES_TEST_DSN or DATABASE_URL.
// Returns "" when no DSN is available (test should skip).
func postgresTestDSN() string {
	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		return dsn
	}

	return os.Getenv("DATABASE_URL")
}

// TestIntegration_PostgresSource_HealthCheck verifies a Postgres-backed
// source-of-truth: dispatch a command, persist an event, load it back,
// run HealthCheck/HealthCheckDetailed, and close cleanly.
//
// Requires a live Postgres instance. Set POSTGRES_TEST_DSN or DATABASE_URL.
// Run via: nix run .#integration-pg -- go test -run TestIntegration_PostgresSource ./system/...
func TestIntegration_PostgresSource_HealthCheck(t *testing.T) {
	dsn := postgresTestDSN()
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN or DATABASE_URL not set; skipping Postgres integration test")
	}

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
								TaskCreated{Title: "pg-integration", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"pg-store": {Driver: "postgres", DSN: dsn},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "pg-store"},
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

	if p.Title != "pg-integration" {
		t.Fatalf("expected title pg-integration, got %s", p.Title)
	}

	// HealthCheck should pass — Postgres implements HealthChecker.
	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// HealthCheckDetailed should include the pg engine and report it healthy.
	detailed := sys.HealthCheckDetailed(ctx)
	found := false

	for _, h := range detailed {
		if h.Error != nil {
			t.Fatalf("engine %s unhealthy: %v", h.Name, h.Error)
		}

		if h.Name == "pg-store" {
			found = true
		}
	}

	if !found {
		t.Fatalf("pg-store not in HealthCheckDetailed: %+v", detailed)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
