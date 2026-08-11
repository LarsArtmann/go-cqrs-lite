package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestIntegration_SQLiteSource_MemoryProjection_HealthCheck verifies a
// two-engine deployment: SQLite for the event source-of-truth, Memory for
// projection read models. Exercises the full lifecycle: construct, dispatch,
// project, HealthCheck, HealthCheckDetailed, GracefulClose.
func TestIntegration_SQLiteSource_MemoryProjection_HealthCheck(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	taskViewQuery := metaengine.Query[FindTask, TaskView](
		"task_views_sqlmem",
		metaengine.OnRecordTyped(
			"task.created",
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (string, TaskView) {
				return e.Title, TaskView{Title: e.Title, Status: "pending"}
			},
		),
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
								TaskCreated{Title: "sqlite-mem-integration", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:       []system.ProjectionDeclaration{system.RawQuery(taskViewQuery)},
		ProjectionDecoder: projectionDecoder,
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"sqlite-store": {
				Driver: "sqlite",
				DSN:    "file:" + t.Name() + "?mode=memory&cache=shared",
			},
			"memory-proj": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "sqlite-store"},
			{Role: system.RoleProjections, Engine: "memory-proj"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for projection to catch up.
	deadline := time.Now().Add(8 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				goto caughtUp
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

caughtUp:
	for _, s := range sys.ProjectionHost().Status() {
		if s.Errors > 0 {
			t.Fatalf("projection %q has %d errors", s.Name, s.Errors)
		}

		if s.Processed < 1 {
			t.Fatalf("projection %q processed %d events, want >= 1", s.Name, s.Processed)
		}
	}

	// Verify projection data.
	result, err := sys.MetaEngine().Execute(FindTask{ID: "sqlite-mem-integration"})
	if err != nil {
		t.Fatalf("MetaEngine.Execute: %v", err)
	}

	view, ok := result.(TaskView)
	if !ok {
		t.Fatalf("expected TaskView, got %T", result)
	}

	if view.Title != "sqlite-mem-integration" || view.Status != "pending" {
		t.Fatalf("unexpected view: %+v", view)
	}

	// HealthCheck should pass on a healthy system.
	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// HealthCheckDetailed should report both engines healthy.
	detailed := sys.HealthCheckDetailed(ctx)
	if len(detailed) == 0 {
		t.Fatal("HealthCheckDetailed returned no engines")
	}

	for _, h := range detailed {
		if h.Error != nil {
			t.Fatalf("engine %s unhealthy: %v", h.Name, h.Error)
		}
	}

	// EngineNames should contain both engine names.
	names := sys.EngineNames()
	if len(names) < 2 {
		t.Fatalf("EngineNames: expected >= 2 engines, got %d (%v)", len(names), names)
	}

	// GracefulClose should drain and close without error.
	gracefulCtx, gracefulCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer gracefulCancel()

	if err := sys.GracefulClose(gracefulCtx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}
}

// TestIntegration_PebbleSource_HealthCheck verifies a Pebble-backed
// source-of-truth with HealthCheck and clean Close.
func TestIntegration_PebbleSource_HealthCheck(t *testing.T) {
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
								TaskCreated{Title: "pebble-integration", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"pebble-store": {Driver: "pebble"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "pebble-store"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)

	if err := sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	// Verify the event was persisted.
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

	if p.Title != "pebble-integration" {
		t.Fatalf("expected title pebble-integration, got %s", p.Title)
	}

	// HealthCheck should pass — Pebble implements HealthChecker.
	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// HealthCheckDetailed should include the pebble engine.
	detailed := sys.HealthCheckDetailed(ctx)
	found := false

	for _, h := range detailed {
		if h.Error != nil {
			t.Fatalf("engine %s unhealthy: %v", h.Name, h.Error)
		}

		if h.Name == "pebble-store" {
			found = true
		}
	}

	if !found {
		t.Fatalf("pebble-store not in HealthCheckDetailed: %+v", detailed)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// eventBusDrainer wraps a Watermill EventBus as a system.Drainer. This is the
// real-world adapter a consumer writes to integrate external async processing
// (notifications, audit logs, etc.) with GracefulClose lifecycle management.
type eventBusDrainer struct {
	bus     *watermill.EventBus
	drained atomic.Bool
}

func (d *eventBusDrainer) Drain(_ context.Context) error {
	d.drained.Store(true)

	return d.bus.Close()
}

// TestIntegration_GracefulClose_WatermillDrainer verifies that GracefulClose
// properly drains a real Watermill EventBus registered as a Drainer before
// closing the system's engines.
func TestIntegration_GracefulClose_WatermillDrainer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx,
		system.DomainConfig{
			Commands: func(sys *system.System) {
				system.RegisterDecider(sys, "Task", TaskDecider)

				system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
					func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
						return system.Execute(ctx, cmd.StreamID(), "Task",
							func(state TaskState, ver event.Version) ([]event.Event, error) {
								return []event.Event{mustEvent(event.New("task.created",
									cmd.StreamID(), "Task", ver+1,
									TaskCreated{Title: "watermill-drain", At: time.Now()},
									event.WithCodec(codec.JSONCodec{})))}, nil
							})
					})
			},
		},
		system.DeploymentConfig{
			Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
			Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
		})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	// Create a real Watermill EventBus (GoChannel-backed).
	wmBus := watermill.NewEventBus()

	// Subscribe before publishing (GoChannel is non-persistent).
	var handlerCalled atomic.Bool

	if err := wmBus.Subscribe("task.created", func(_ context.Context, evt event.Event) error {
		var p TaskCreated
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return err
		}

		if p.Title != "watermill-drain" {
			t.Errorf("expected title watermill-drain, got %s", p.Title)
		}

		handlerCalled.Store(true)

		return nil
	}); err != nil {
		t.Fatalf("wmBus.Subscribe: %v", err)
	}

	// Register the EventBus as a Drainer so GracefulClose shuts it down.
	drainer := &eventBusDrainer{bus: wmBus}
	sys.RegisterDrainer(drainer)

	// Publish a real event through the Watermill EventBus.
	streamID := id.NewStreamID()
	evt := mustEvent(event.New("task.created", streamID, "Task", 1,
		TaskCreated{Title: "watermill-drain", At: time.Now()},
		event.WithCodec(codec.JSONCodec{})))

	if err := wmBus.Publish(ctx, evt); err != nil {
		t.Fatalf("wmBus.Publish: %v", err)
	}

	// BlockPublishUntilSubscriberAck guarantees the handler ran during Publish.
	if !handlerCalled.Load() {
		t.Fatal("Watermill handler was not called")
	}

	// GracefulClose should drain the drainer then close the system.
	if err := sys.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	// Verify the drainer was called during GracefulClose.
	if !drainer.drained.Load() {
		t.Fatal("drainer.Drain was not called during GracefulClose")
	}

	// Verify the EventBus is now closed (Publish returns error).
	if err := wmBus.Publish(ctx, evt); err == nil {
		t.Fatal("expected error publishing to closed EventBus")
	}
}
