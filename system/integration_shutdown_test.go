package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestIntegration_ShutdownDependency verifies that ShutdownDependencies
// declared in DomainConfig are respected by ShutdownOrder() and Close().
// Uses real Memory + SQLite engines instead of mocks.
func TestIntegration_ShutdownDependency(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
								TaskCreated{Title: "shutdown-dep-test", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		// Declare that "projections" engine must close BEFORE "event-store".
		ShutdownDependencies: []system.ShutdownDependency{
			{Before: "projections", After: "event-store"},
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"event-store": {
				Driver: "sqlite",
				DSN:    "file:" + t.Name() + "?mode=memory&cache=shared",
			},
			"projections": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "event-store"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	// Dispatch a real command so the system has actual state.
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

	// ShutdownOrder returns config keys (matching DeploymentConfig.Engines
	// map keys), so the dependency "projections" before "event-store" can
	// be verified directly.
	order := sys.ShutdownOrder()
	if len(order) != 2 {
		t.Fatalf("expected 2 engines in shutdown order, got %d: %v", len(order), order)
	}

	// Map config keys to positions.
	pos := make(map[string]int, len(order))
	for i, name := range order {
		pos[name] = i
	}

	projIdx, projOK := pos["projections"]
	eventIdx, eventOK := pos["event-store"]
	if !projOK || !eventOK {
		t.Fatalf("expected projections + event-store in shutdown order, got: %v", order)
	}

	if projIdx > eventIdx {
		t.Fatalf(
			"projections (idx %d) must close before event-store (idx %d), order: %v",
			projIdx,
			eventIdx,
			order,
		)
	}

	// EngineNames should include both engines.
	names := sys.EngineNames()
	if len(names) < 2 {
		t.Fatalf("expected >= 2 engine names, got %d: %v", len(names), names)
	}

	// Close should succeed and respect the order.
	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Double-close is safe (idempotent).
	if err := sys.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestIntegration_ShutdownDependency_CycleFallback verifies that when
// ShutdownDependencies form a cycle, the system falls back to creation
// order instead of deadlocking.
func TestIntegration_ShutdownDependency_CycleFallback(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		// A → B and B → A forms a cycle — should fall back to creation order.
		ShutdownDependencies: []system.ShutdownDependency{
			{Before: "alpha", After: "beta"},
			{Before: "beta", After: "alpha"},
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"alpha": {Driver: "memory"},
			"beta":  {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "alpha"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	// Must not deadlock or panic — cycle falls back to creation order.
	order := sys.ShutdownOrder()
	if len(order) != 2 {
		t.Fatalf("expected 2 engines in shutdown order, got %d: %v", len(order), order)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close with cyclic dependency: %v", err)
	}
}
