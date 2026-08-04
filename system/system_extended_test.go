package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Query dispatch test ──

type countQuery struct {
	Filter string
}

func (countQuery) Type() query.Type { return query.Type("count") }

type countResult struct {
	Count int
}

func TestSystem_QueryDispatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Queries: func(sys *system.System) {
			system.RegisterQuery[countQuery, countResult](sys, "count",
				func(_ context.Context, q countQuery) (countResult, error) {
					return countResult{Count: 42}, nil
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

	result, err := system.DispatchQuery[countQuery, countResult](
		ctx,
		sys,
		countQuery{Filter: "all"},
	)
	if err != nil {
		t.Fatalf("DispatchQuery: %v", err)
	}

	if result.Count != 42 {
		t.Fatalf("expected count 42, got %d", result.Count)
	}
}

// ── Driver registry test ──

func TestSystem_DriverRegistry(t *testing.T) {
	t.Parallel()

	drivers := system.RegisteredDrivers()
	found := false

	for _, d := range drivers {
		if d == "memory" {
			found = true
		}
	}

	if !found {
		t.Fatal("memory driver not registered")
	}
}

// ── Snapshot backend test ──

func TestSnapshotBackend_MemoryRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	// The memory engine doesn't implement SnapshotBackend directly — the system
	// package's memorySnapshotBackend does. Test it directly.
	backend := system.NewMemorySnapshotBackend()

	data := []byte(`{"title":"test","status":"pending"}`)

	if err := backend.SnapshotSave(ctx, "snapshots", "stream-1", 5, data); err != nil {
		t.Fatalf("SnapshotSave: %v", err)
	}

	loaded, version, err := backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if err != nil {
		t.Fatalf("SnapshotLoad: %v", err)
	}

	if version != 5 {
		t.Fatalf("expected version 5, got %d", version)
	}

	if string(loaded) != string(data) {
		t.Fatalf("data mismatch: got %s", loaded)
	}

	// LoadAtVersion with matching version.
	loadedAt, _, err := backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 5)
	if err != nil {
		t.Fatalf("SnapshotLoadAtVersion: %v", err)
	}

	if string(loadedAt) != string(data) {
		t.Fatalf("data mismatch at version")
	}

	// LoadAtVersion with version too low.
	_, _, err = backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 3)
	if err == nil {
		t.Fatal("expected error for version too low")
	}

	// Delete.
	if err := backend.SnapshotDelete(ctx, "snapshots", "stream-1"); err != nil {
		t.Fatalf("SnapshotDelete: %v", err)
	}

	_, _, err = backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// ── Multi-decider test ──

type CounterState struct {
	Count int
}

type CounterIncremented struct {
	Delta int
}

func applyCounter(state CounterState, evt event.Event) (CounterState, error) {
	if evt.Type() == "counter.incremented" {
		var p CounterIncremented
		_ = json.Unmarshal(evt.Payload(), &p)
		state.Count += p.Delta
	}

	return state, nil
}

var CounterDecider = decider.Decider[CounterState]{
	Initial: CounterState{},
	Apply:   applyCounter,
}

func TestSystem_MultiDecider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)
			system.RegisterDecider(sys, "Counter", CounterDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "multi", At: time.Now()}))}, nil
						})
				})

			system.RegisterCommand[*command.BasicCommand, CounterState](sys, "counter.increment",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[CounterState] {
					return system.Execute(ctx, cmd.StreamID(), "Counter",
						func(state CounterState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("counter.incremented",
								cmd.StreamID(), "Counter", ver+1,
								CounterIncremented{Delta: 5}))}, nil
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

	// Create a task.
	taskID := id.NewStreamID()
	_ = sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", taskID))

	// Increment a counter.
	counterID := id.NewStreamID()
	_ = sys.CommandDispatcher().Dispatch(ctx, newCmd("counter.increment", counterID))

	// Verify both streams.
	taskEvents, _ := sys.EventStore().Load(ctx, id.NewStreamRef("Task", taskID))
	if len(taskEvents) != 1 {
		t.Fatalf("expected 1 task event, got %d", len(taskEvents))
	}

	counterEvents, _ := sys.EventStore().Load(ctx, id.NewStreamRef("Counter", counterID))
	if len(counterEvents) != 1 {
		t.Fatalf("expected 1 counter event, got %d", len(counterEvents))
	}
}

// ── Concurrent command dispatch test (race detector) ──

func TestSystem_ConcurrentDispatch(t *testing.T) {
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
								TaskCreated{Title: "concurrent", At: time.Now()}))}, nil
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

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			streamID := id.NewStreamID()
			_ = sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID))
		}()
	}

	wg.Wait()

	// Verify all 20 tasks were created.
	journal := sys.EventStore().(event.Journal)
	all, _ := journal.ReadAll(ctx)
	if len(all) != 20 {
		t.Fatalf("expected 20 events, got %d", len(all))
	}
}

// ── Event bus pub/sub test ──

func TestSystem_EventBusPubSub(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var received atomic.Int32

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "bus test", At: time.Now()}))}, nil
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

	// Subscribe before dispatching.
	_ = sys.Bus().Subscribe("task.created", func(_ context.Context, evt event.Event) error {
		received.Add(1)

		return nil
	})

	streamID := id.NewStreamID()
	if err := sys.CommandDispatcher().Dispatch(ctx, newCmd("task.create", streamID)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if received.Load() != 1 {
		t.Fatalf("expected 1 received event, got %d", received.Load())
	}
}

// ── MultiBus fan-out test ──

func TestMultiBus_FanOut(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var count1, count2 atomic.Int32

	bus1 := event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		count1.Add(1)

		return nil
	})
	bus2 := event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		count2.Add(1)

		return nil
	})

	multi := system.NewMultiBus(bus1, bus2)

	evt := mustEvent(
		event.New("test.event", id.NewStreamID(), "Test", 1, map[string]string{"k": "v"}),
	)
	if err := multi.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if count1.Load() != 1 || count2.Load() != 1 {
		t.Fatalf(
			"expected both buses to receive event: bus1=%d, bus2=%d",
			count1.Load(),
			count2.Load(),
		)
	}
}

// ── SnapshotBackend isolation test ──

func TestSnapshotBackend_Isolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	b1 := system.NewMemorySnapshotBackend()
	b2 := system.NewMemorySnapshotBackend()

	_ = b1.SnapshotSave(ctx, "snapshots", "stream-1", 1, []byte("data-1"))
	_ = b2.SnapshotSave(ctx, "snapshots", "stream-1", 1, []byte("data-2"))

	data1, _, _ := b1.SnapshotLoad(ctx, "snapshots", "stream-1")
	data2, _, _ := b2.SnapshotLoad(ctx, "snapshots", "stream-1")

	if string(data1) != "data-1" {
		t.Fatalf("b1 data mismatch: got %s", data1)
	}

	if string(data2) != "data-2" {
		t.Fatalf("b2 data mismatch: got %s", data2)
	}
}

// ── Op accessor test ──

func TestOp_Accessors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	streamID := id.NewStreamID()
	streamType := id.StreamType("Task")

	decideFn := func(state TaskState, ver event.Version) ([]event.Event, error) { return nil, nil }

	op := system.Execute(ctx, streamID, streamType, decideFn)

	if op.StreamID() != streamID {
		t.Fatalf("StreamID mismatch")
	}

	if op.StreamType() != streamType {
		t.Fatalf("StreamType mismatch")
	}
}

// ── AtomicAppender conflict test (concurrent writes to same stream) ──

func TestSystem_AtomicConcurrencyConflict(t *testing.T) {
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
								TaskCreated{Title: "race", At: time.Now()}))}, nil
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

	// Direct Save with wrong version should fail.
	ref := id.NewStreamRef("Task", id.NewStreamID())
	err = sys.EventStore().Save(ctx, ref,
		[]event.Event{
			mustEvent(
				event.New(
					"task.created",
					ref.ID,
					"Task",
					1,
					TaskCreated{Title: "x", At: time.Now()},
				),
			),
		},
		event.Version(0))
	if err != nil {
		t.Fatalf("first Save should succeed: %v", err)
	}

	// Second Save with same expected version (0) should conflict.
	err = sys.EventStore().Save(ctx, ref,
		[]event.Event{
			mustEvent(
				event.New("task.completed", ref.ID, "Task", 2, TaskCompleted{At: time.Now()}),
			),
		},
		event.Version(0))

	if err == nil {
		t.Fatal("expected version conflict on second Save")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}
