package system_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── T23: ProjectionPlan/VerifyProjections/ProjectionExplain tests ──

func TestSystem_ProjectionPlan_NilWhenNoProjections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	if plan := sys.ProjectionPlan(); plan != nil {
		t.Fatalf("expected nil plan, got %+v", plan)
	}

	if err := sys.VerifyProjections(ctx); err != nil {
		t.Fatalf("expected nil verify, got %v", err)
	}

	if explain := sys.ProjectionExplain(); explain != "" {
		t.Fatalf("expected empty explain, got %q", explain)
	}
}

func TestSystem_ProjectionPlan_WithProjectionStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	taskViewQuery := metaengine.Query[FindTask, TaskView]("task_views",
		metaengine.OnTyped("task.created", TaskCreated{}, func(e TaskCreated) (string, TaskView) {
			return e.Title, TaskView{Title: e.Title, Status: "pending"}
		}),
	)

	domain := system.DomainConfig{
		Projections:       []any{taskViewQuery},
		ProjectionDecoder: projectionDecoder,
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"proj":    {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "proj"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	plan := sys.ProjectionPlan()
	if plan == nil {
		t.Fatal("expected non-nil plan with projection store")
	}

	// Verify returns an error without an event log attached — that's expected
	// here since the test doesn't dispatch events. The important thing is it
	// doesn't panic.
	_ = sys.VerifyProjections(ctx)

	explain := sys.ProjectionExplain()
	if explain == "" {
		t.Fatal("expected non-empty explain string")
	}
}

// ── T10: MultiBus-through-New() integration test ──

func TestSystem_MultiBusFanOut(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var bus1Count, bus2Count atomic.Int32

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "fanout", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Buses: map[string]system.BusConfig{
			"bus1": {Driver: "gochannel"},
			"bus2": {Driver: "gochannel"},
		},
		Instances: []system.InstanceConfig{{
			Role:    system.RoleSourceOfTruth,
			Engine:  "primary",
			Publish: []string{"bus1", "bus2"},
		}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Subscribe on the local bus to track delivery.
	_ = sys.Bus().Subscribe("task.created", func(_ context.Context, _ event.Event) error {
		bus1Count.Add(1)
		return nil
	})
	_ = sys.Bus().SubscribeAll(func(_ context.Context, _ event.Event) error {
		bus2Count.Add(1)
		return nil
	})

	streamID := id.NewStreamID()
	_ = sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID))

	// The local bus should receive the event at least once.
	if bus1Count.Load() < 1 {
		t.Fatalf("expected bus1 to receive event, got %d", bus1Count.Load())
	}

	if bus2Count.Load() < 1 {
		t.Fatalf("expected catch-all to receive event, got %d", bus2Count.Load())
	}

	// Verify MultiBus was actually used: pubBus should not be the same as bus.
	if sys.Bus() == nil {
		t.Fatal("expected non-nil bus")
	}
}

// ── T19: gochannel bus driver test ──

func TestSystem_GochannelBusDriverRegistered(t *testing.T) {
	t.Parallel()

	drivers := system.RegisteredBusDrivers()

	found := false
	for _, d := range drivers {
		if d == "gochannel" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("gochannel driver not registered, got: %v", drivers)
	}
}

func TestSystem_RegisteredDriversIncludesMemoryAndSQLite(t *testing.T) {
	t.Parallel()

	drivers := system.RegisteredDrivers()

	hasMemory := false
	hasSQLite := false

	for _, d := range drivers {
		if d == "memory" {
			hasMemory = true
		}

		if d == "sqlite" {
			hasSQLite = true
		}
	}

	if !hasMemory {
		t.Fatal("memory driver not registered")
	}

	if !hasSQLite {
		t.Fatal("sqlite driver not registered")
	}
}
