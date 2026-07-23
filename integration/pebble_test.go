package integration_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

// counterState is a tiny aggregate state for the pebble integration tests.
type counterState struct {
	Value int
}

func applyCounter(s counterState, evt event.Event) (counterState, error) {
	switch evt.Type() {
	case "CounterIncremented":
		return counterState{Value: s.Value + 1}, nil
	default:
		return s, nil
	}
}

func TestPebbleEventStoreWithProjectionRunner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	store := backend.EventStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	// Seed two events before the runner starts so replay is exercised.
	aggID := id.NewStreamID()
	ref := id.NewStreamRef(id.StreamType("Counter"), aggID)

	for i := range 2 {
		version := event.Version(i + 1)
		evt, err := event.NewEvent(
			"CounterIncremented",
			aggID,
			id.StreamType("Counter"),
			version,
			nil,
		)
		if err != nil {
			t.Fatalf("new event: %v", err)
		}

		if err := store.Save(ctx, ref, []event.Event{evt}, version-1); err != nil {
			t.Fatalf("save event: %v", err)
		}
	}

	var received int

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error {
		received++

		return nil
	})

	time.Sleep(50 * time.Millisecond)

	// Emit a live event through the bus (not the store) to test live subscription.
	liveEvt, err := event.NewEvent(
		"CounterIncremented",
		aggID,
		id.StreamType("Counter"),
		event.Version(3),
		nil,
	)
	if err != nil {
		t.Fatalf("new live event: %v", err)
	}

	if err := bus.Publish(ctx, liveEvt); err != nil {
		t.Fatalf("publish live event: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Only the live event is received (bus.SubscribeAll does not replay journal).
	if received != 1 {
		t.Fatalf("expected 1 live event, got %d", received)
	}
}

func TestPebbleSnapshotStoreWithDeciderRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	store := backend.EventStore()
	snapStore := backend.SnapshotStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	strategy, err := snapshot.EveryNEvents(2)
	if err != nil {
		t.Fatalf("new strategy: %v", err)
	}

	repo, err := decider.NewRepository(
		store, bus, d,
		decider.WithSnapshotStore[counterState](snapStore),
		decider.WithCodec[counterState](codec.JSONCodec{}),
		decider.WithSnapshotStrategy[counterState](strategy),
	)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	aggID := id.NewStreamID()
	ref := id.NewStreamRef(id.StreamType("Counter"), aggID)

	// Execute three increments; snapshot should be saved at version 2.
	for range 3 {
		if err := repo.Execute(
			ctx,
			aggID,
			"Counter",
			func(s counterState, v event.Version) ([]event.Event, error) {
				evt, err := event.NewEvent(
					"CounterIncremented",
					aggID,
					id.StreamType("Counter"),
					v.Add(1),
					nil,
				)
				if err != nil {
					return nil, err
				}

				return []event.Event{evt}, nil
			},
		); err != nil {
			t.Fatalf("execute: %v", err)
		}
	}

	// Load directly through snapshot store to verify a snapshot was persisted.
	snap, err := snapStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	if snap.Version != event.Version(2) {
		t.Fatalf("expected snapshot at version 2, got %d", snap.Version)
	}

	// Load aggregate through repository: it should use the snapshot and only load
	// events after the snapshot version.
	state, _, err := repo.Load(ctx, aggID, "Counter")
	if err != nil {
		t.Fatalf("load aggregate: %v", err)
	}

	if state.Value != 3 {
		t.Fatalf("expected aggregate value 3, got %d", state.Value)
	}
}
