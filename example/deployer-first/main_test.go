package main

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// TestDeployerFirst_FullPipeline proves the async pipeline produces the
// correct materialized view with ordered event processing: create → complete →
// delete (tombstone). Commands execute before the projection starts; the
// CatchUpSubscriber replays from the journal (ordered).
func TestDeployerFirst_FullPipeline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bundle := newTestBundle(t)
	mat := newTestMaterialize(t, bundle)
	repo := newTestRepo(t, bundle)

	todoID := id.NewAggregateID()

	if err := repo.Execute(
		ctx,
		todoID,
		aggregateType,
		decideCreate(todoID, "Write tests"),
	); err != nil {
		t.Fatal(err)
	}

	if err := repo.Execute(ctx, todoID, aggregateType, decideComplete(todoID)); err != nil {
		t.Fatal(err)
	}

	if err := repo.Execute(ctx, todoID, aggregateType, decideDelete(todoID, "done")); err != nil {
		t.Fatal(err)
	}

	startProjection(t, ctx, bundle, mat)

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if v, err := mat.View(ctx, todoID); err == nil &&
			v.Title == "Write tests" && v.Completed && v.Tombstoned {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	v, _ := mat.View(ctx, todoID)
	t.Fatalf("timed out; view=%+v", v)
}

// TestDeployerFirst_LiveDelivery tests events published AFTER the projection
// starts are delivered in order via the live path (not replay).
func TestDeployerFirst_LiveDelivery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bundle := newTestBundle(t)
	mat := newTestMaterialize(t, bundle)
	repo := newTestRepo(t, bundle)

	startProjection(t, ctx, bundle, mat)

	time.Sleep(100 * time.Millisecond) // let live phase subscribe

	todoID := id.NewAggregateID()

	if err := repo.Execute(ctx, todoID, aggregateType, decideCreate(todoID, "Live")); err != nil {
		t.Fatal(err)
	}

	if err := repo.Execute(ctx, todoID, aggregateType, decideComplete(todoID)); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if v, err := mat.View(ctx, todoID); err == nil &&
			v.Title == "Live" && v.Completed {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	v, _ := mat.View(ctx, todoID)
	t.Fatalf("timed out; view=%+v", v)
}

// helpers

func newTestBundle(t *testing.T) *stack.Bundle {
	t.Helper()

	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	t.Cleanup(func() { _ = bundle.Close() })

	return bundle
}

func newTestMaterialize(
	t *testing.T,
	bundle *stack.Bundle,
) *stack.Materialize[TodoView, id.AggregateID] {
	t.Helper()

	mat, err := stack.NewMaterialize[TodoView, id.AggregateID](bundle, codec.JSONCodec{}, todoKey)
	if err != nil {
		t.Fatalf("NewMaterialize: %v", err)
	}

	configureMaterialize(mat)

	return mat
}

func newTestRepo(t *testing.T, bundle *stack.Bundle) *decider.Repository[TodoState] {
	t.Helper()

	repo, err := stack.Repository[TodoState](bundle, decider.Decider[TodoState]{
		Initial: TodoState{},
		Apply:   applyTodo,
	})
	if err != nil {
		t.Fatalf("Repository: %v", err)
	}

	return repo
}

// startProjection creates a CatchUpSubscriber and consumes its output channel
// in a single goroutine, calling the Materialize handler for each message.
// Single-goroutine consumption guarantees FIFO ordering.
func startProjection(
	t *testing.T,
	ctx context.Context,
	bundle *stack.Bundle,
	mat *stack.Materialize[TodoView, id.AggregateID],
) {
	t.Helper()

	catchUp, err := bundle.CatchUpSubscriber()
	if err != nil {
		t.Fatalf("CatchUpSubscriber: %v", err)
	}

	msgs, err := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	handler := mat.HandlerFunc()
	go func() {
		for msg := range msgs {
			_ = handler(msg)
			msg.Ack()
		}
	}()
}

var _ event.Type // keep event import
