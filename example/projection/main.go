// Package main demonstrates building and running projections using the projection module.
//
// It shows:
//  1. Creating an in-memory event store and bus
//  2. Building a type-safe projection with projection.On[T]
//  3. Running replay + live subscription via projection.Runner
//
// Run with: go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

type ItemAdded struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type ItemRemoved struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type InventoryReadModel struct {
	Items map[string]int
}

func newInventoryReadModel() *InventoryReadModel {
	return &InventoryReadModel{Items: make(map[string]int)}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	readModel := newInventoryReadModel()

	builder := projection.NewBuilder("inventory-projection")
	if err := projection.On[ItemAdded](builder, "item.added", func(_ context.Context, payload ItemAdded) error {
		readModel.Items[payload.Name] += payload.Quantity
		fmt.Printf("  [projection] added %d %s (total: %d)\n", payload.Quantity, payload.Name, readModel.Items[payload.Name])
		return nil
	}); err != nil {
		log.Fatalf("register item.added: %v", err)
	}
	if err := projection.On[ItemRemoved](builder, "item.removed", func(_ context.Context, payload ItemRemoved) error {
		readModel.Items[payload.Name] -= payload.Quantity
		fmt.Printf("  [projection] removed %d %s (total: %d)\n", payload.Quantity, payload.Name, readModel.Items[payload.Name])
		return nil
	}); err != nil {
		log.Fatalf("register item.removed: %v", err)
	}

	inventoryProjection := builder.Build()

	checkpointStore := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(store, bus, checkpointStore)
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}
	if err := runner.Register(inventoryProjection); err != nil {
		log.Fatalf("register projection: %v", err)
	}

	go func() {
		if runErr := runner.Run(ctx); runErr != nil {
			log.Printf("runner stopped: %v", runErr)
		}
	}()

	fmt.Println("Saving historical events...")
	aggregateID := id.NewAggregateID()
	for i, evt := range []struct {
		eventType event.Type
		payload   any
	}{
		{"item.added", ItemAdded{Name: "widget", Quantity: 10}},
		{"item.added", ItemAdded{Name: "gadget", Quantity: 5}},
		{"item.removed", ItemAdded{Name: "widget", Quantity: 3}},
	} {
		payload, _ := json.Marshal(evt.payload)
		e, createErr := event.NewEvent(evt.eventType, aggregateID, "Inventory", event.Version(i+1), payload)
		if createErr != nil {
			log.Fatalf("create event: %v", createErr)
		}
		if saveErr := store.Save(ctx, "Inventory", aggregateID, []event.Event{e}, event.Version(i)); saveErr != nil {
			log.Fatalf("save event: %v", saveErr)
		}
		if pubErr := bus.Publish(ctx, e); pubErr != nil {
			log.Fatalf("publish event: %v", pubErr)
		}
	}

	time.Sleep(100 * time.Millisecond)

	fmt.Println("\nRead model state:")
	for name, qty := range readModel.Items {
		fmt.Printf("  %s: %d\n", name, qty)
	}

	_ = runner.Close()
	fmt.Println("\nDone.")
}
