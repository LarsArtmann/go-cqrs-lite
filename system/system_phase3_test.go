package system_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestSimpleBus_HandlerIndependence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "indep", At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	var handler1Called, handler2Called atomic.Int32

	// First handler returns an error.
	_ = sys.Bus().Subscribe("task.created", func(_ context.Context, _ event.Event) error {
		handler1Called.Add(1)

		return errors.New("handler 1 failed")
	})

	// Second handler should still execute despite first handler's error.
	_ = sys.Bus().Subscribe("task.created", func(_ context.Context, _ event.Event) error {
		handler2Called.Add(1)

		return nil
	})

	streamID := id.NewStreamID()
	_ = sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID))

	if handler1Called.Load() != 1 {
		t.Fatalf("expected handler1 called once, got %d", handler1Called.Load())
	}

	if handler2Called.Load() != 1 {
		t.Fatalf("expected handler2 called once (independent), got %d", handler2Called.Load())
	}
}

func TestSystem_IntrospectionRealValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)
			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "intro", At: time.Now()}))}, nil
						})
				})
		},
	}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	topo, err := sys.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	// Health status should not be hardcoded "ok" — it should reflect real health.
	for _, inst := range topo.Instances {
		if inst.HealthStatus == "" {
			t.Fatalf("HealthStatus is empty for instance %s", inst.Name)
		}
	}

	// Handler count should reflect actual registered handlers.
	var foundCommand bool

	for _, d := range topo.Dispatchers {
		if d.Type == "command" {
			foundCommand = true
			if d.Handlers != 1 {
				t.Fatalf("expected 1 command handler, got %d", d.Handlers)
			}
		}
	}

	if !foundCommand {
		t.Fatal("command dispatcher not found in topology")
	}
}

func TestSystem_ScreamStoreWarnsOnMemorySOT(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Memory engine as source-of-truth should generate a WARN+OVERRIDE diagnostic.
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	report, err := system.CheckSafety(ctx, deployment)
	if err != nil {
		t.Fatalf("CheckSafety: %v", err)
	}

	if !report.HasWarnings() {
		t.Fatal("expected warnings for memory source-of-truth")
	}

	foundVolatileRule := false

	for _, d := range report.Diagnostics {
		if d.Rule == "volatile-source-of-truth" {
			foundVolatileRule = true
		}
	}

	if !foundVolatileRule {
		t.Fatal("expected volatile-source-of-truth warning")
	}
}
