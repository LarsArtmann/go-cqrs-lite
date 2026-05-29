package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func TestProjection_ReplayAndLive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	readModel := newInventoryReadModel()

	builder := projection.NewBuilder("inventory-test")
	if err := projection.On[ItemAdded](
		builder,
		"item.added",
		codec.JSONCodec{},
		func(_ context.Context, payload ItemAdded) error {
			readModel.Items[payload.Name] += payload.Quantity
			return nil
		},
	); err != nil {
		t.Fatalf("register item.added: %v", err)
	}

	if err := projection.On[ItemRemoved](
		builder,
		"item.removed",
		codec.JSONCodec{},
		func(_ context.Context, payload ItemRemoved) error {
			readModel.Items[payload.Name] -= payload.Quantity
			return nil
		},
	); err != nil {
		t.Fatalf("register item.removed: %v", err)
	}

	inventoryProjection := builder.Build()
	checkpointStore := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(store, bus, checkpointStore)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}

	if err := runner.Register(inventoryProjection); err != nil {
		t.Fatalf("register projection: %v", err)
	}

	go func() {
		_ = runner.Run(ctx)
	}()

	aggregateID := id.NewAggregateID()

	events := []struct {
		eventType event.Type
		payload   any
	}{
		{"item.added", ItemAdded{Name: "widget", Quantity: 10}},
		{"item.added", ItemAdded{Name: "gadget", Quantity: 5}},
		{"item.removed", ItemRemoved{Name: "widget", Quantity: 3}},
	}

	for i, evt := range events {
		payloadBytes, _ := json.Marshal(evt.payload)

		e, createErr := event.NewEvent(
			evt.eventType, aggregateID, "Inventory", event.Version(i+1), payloadBytes,
		)
		if createErr != nil {
			t.Fatalf("create event %d: %v", i, createErr)
		}

		if saveErr := store.Save(
			ctx,
			event.NewAggregateRef("Inventory", aggregateID),
			[]event.Event{e},
			event.Version(i),
		); saveErr != nil {
			t.Fatalf("save event %d: %v", i, saveErr)
		}

		if pubErr := bus.Publish(ctx, e); pubErr != nil {
			t.Fatalf("publish event %d: %v", i, pubErr)
		}
	}

	time.Sleep(200 * time.Millisecond)
	_ = runner.Close()

	if readModel.Items["widget"] != 7 {
		t.Errorf("widget = %d, want 7", readModel.Items["widget"])
	}

	if readModel.Items["gadget"] != 5 {
		t.Errorf("gadget = %d, want 5", readModel.Items["gadget"])
	}
}
