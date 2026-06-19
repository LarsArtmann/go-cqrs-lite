package projection_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// controllableLeader is a LeaderElection implementation whose leadership
// state can be toggled at runtime for testing.
type controllableLeader struct {
	leader       atomic.Bool
	waitCalled   atomic.Bool
	resignCalled atomic.Bool
	waitErr      error
}

func (c *controllableLeader) IsLeader(_ context.Context) bool {
	return c.leader.Load()
}

func (c *controllableLeader) WaitForLeadership(_ context.Context) error {
	c.waitCalled.Store(true)
	if c.waitErr != nil {
		return c.waitErr
	}

	c.leader.Store(true)

	return nil
}

func (c *controllableLeader) Resign(_ context.Context) error {
	c.resignCalled.Store(true)
	c.leader.Store(false)

	return nil
}

func newDistTestRunner(t *testing.T) (*projection.Runner, *memory.MemoryStore, *memory.MemoryBus) {
	t.Helper()

	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	bus := memory.NewMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, store, bus
}

type countingProjection struct {
	name   string
	count  atomic.Int64
	events []event.Type
}

func (p *countingProjection) Name() string { return p.name }

func (p *countingProjection) Handle(_ context.Context, evt event.Event) error {
	p.count.Add(1)

	return nil
}

func (p *countingProjection) EventTypes() []event.Type { return p.events }

func TestDistributedRunner_AlwaysLeader(t *testing.T) {
	t.Parallel()

	runner, store, _ := newDistTestRunner(t)

	aggID := id.NewAggregateID()
	evt := eventtest.NewEventOpts(t, "test.created", aggID, "Test", 1, nil)

	if err := store.Save(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	proj := &countingProjection{name: "counter", events: []event.Type{"test.created"}}
	if err := runner.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	dr, err := projection.NewDistributedRunner(runner, projection.AlwaysLeader{},
		projection.WithLeadershipCheckInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewDistributedRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = dr.Run(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run: %v", err)
	}

	if proj.count.Load() < 1 {
		t.Errorf("expected projection to process at least 1 event, got %d", proj.count.Load())
	}
}

func TestDistributedRunner_LeadershipLost(t *testing.T) {
	t.Parallel()

	runner, store, _ := newDistTestRunner(t)

	aggID := id.NewAggregateID()
	evt := eventtest.NewEventOpts(t, "test.created", aggID, "Test", 1, nil)

	if err := store.Save(context.Background(),
		event.NewAggregateRef("Test", aggID), []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	proj := &countingProjection{name: "counter", events: []event.Type{"test.created"}}
	if err := runner.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	leader := &controllableLeader{}

	dr, err := projection.NewDistributedRunner(runner, leader,
		projection.WithLeadershipCheckInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("NewDistributedRunner: %v", err)
	}

	runDone := make(chan error, 1)

	go func() {
		runDone <- dr.Run(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	leader.leader.Store(false)

	select {
	case err := <-runDone:
		if !errors.Is(err, projection.ErrLeadershipLost) {
			t.Errorf("expected ErrLeadershipLost, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after leadership loss")
	}

	if !leader.resignCalled.Load() {
		t.Error("expected Resign to be called")
	}
}

func TestDistributedRunner_NilRunner(t *testing.T) {
	t.Parallel()

	_, err := projection.NewDistributedRunner(nil, projection.AlwaysLeader{})
	if err == nil {
		t.Fatal("expected error for nil runner")
	}
}

func TestDistributedRunner_NilElection(t *testing.T) {
	t.Parallel()

	runner, _, _ := newDistTestRunner(t)

	_, err := projection.NewDistributedRunner(runner, nil)
	if err == nil {
		t.Fatal("expected error for nil leader election")
	}
}

func TestDistributedRunner_WaitForLeadershipFails(t *testing.T) {
	t.Parallel()

	runner, _, _ := newDistTestRunner(t)

	waitErr := errors.New("election unavailable")
	leader := &controllableLeader{waitErr: waitErr}

	dr, err := projection.NewDistributedRunner(runner, leader)
	if err != nil {
		t.Fatalf("NewDistributedRunner: %v", err)
	}

	err = dr.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when WaitForLeadership fails")
	}
}

func TestDistributedRunner_RunnerAccessor(t *testing.T) {
	t.Parallel()

	runner, _, _ := newDistTestRunner(t)

	dr, err := projection.NewDistributedRunner(runner, projection.AlwaysLeader{})
	if err != nil {
		t.Fatalf("NewDistributedRunner: %v", err)
	}

	if dr.Runner() != runner {
		t.Error("Runner() should return the underlying runner")
	}
}
