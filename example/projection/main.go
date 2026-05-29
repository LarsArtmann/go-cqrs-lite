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

const (
	contextTimeout = 5 * time.Second
	widgetQty      = 10
	gadgetQty      = 5
	removeQty      = 3
	settleDelay    = 100 * time.Millisecond
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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	readModel := newInventoryReadModel()

	builder := projection.NewBuilder("inventory-projection")

	applyInventory := func(name string, quantity int, verb string, op func(int, int) int) {
		readModel.Items[name] = op(readModel.Items[name], quantity)
		fmt.Printf("  [projection] %s %d %s (total: %d)\n", verb, quantity, name, readModel.Items[name])
	}

	if err := projection.On[ItemAdded](
		builder,
		"item.added",
		func(_ context.Context, p ItemAdded) error {
			applyInventory(p.Name, p.Quantity, "added", func(a, b int) int { return a + b })
			return nil
		},
	); err != nil {
		return fmt.Errorf("register item.added: %w", err)
	}

	if err := projection.On[ItemRemoved](
		builder,
		"item.removed",
		func(_ context.Context, p ItemRemoved) error {
			applyInventory(p.Name, p.Quantity, "removed", func(a, b int) int { return a - b })
			return nil
		},
	); err != nil {
		return fmt.Errorf("register item.removed: %w", err)
	}

	inventoryProjection := builder.Build()

	checkpointStore := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(store, bus, checkpointStore)
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}

	if err := runner.Register(inventoryProjection); err != nil {
		return fmt.Errorf("register projection: %w", err)
	}

	go func() {
		if runErr := runner.Run(ctx); runErr != nil {
			log.Printf("runner stopped: %v", runErr)
		}
	}()

	fmt.Println("Saving historical events...")

	aggregateID := id.NewAggregateID()

	events := []struct {
		eventType event.Type
		name      string
		quantity  int
	}{
		{"item.added", "widget", widgetQty},
		{"item.added", "gadget", gadgetQty},
		{"item.removed", "widget", removeQty},
	}

	for i, evt := range events {
		payload := ItemAdded{Name: evt.name, Quantity: evt.quantity}

		payloadBytes, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("marshal event %d: %w", i, marshalErr)
		}

		e, createErr := event.NewEvent(
			evt.eventType,
			aggregateID,
			"Inventory",
			event.Version(i+1),
			payloadBytes,
		)
		if createErr != nil {
			return fmt.Errorf("create event %d: %w", i, createErr)
		}

		if saveErr := store.Save(
			ctx,
			"Inventory",
			aggregateID,
			[]event.Event{e},
			event.Version(i),
		); saveErr != nil {
			return fmt.Errorf("save event %d: %w", i, saveErr)
		}

		if pubErr := bus.Publish(ctx, e); pubErr != nil {
			return fmt.Errorf("publish event %d: %w", i, pubErr)
		}
	}

	time.Sleep(settleDelay)

	fmt.Println("\nRead model state:")

	for name, qty := range readModel.Items {
		fmt.Printf("  %s: %d\n", name, qty)
	}

	_ = runner.Close()

	fmt.Println("\nDone.")

	return nil
}
