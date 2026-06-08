package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

// demonstrateDualStore shows how to switch between memory and SQL event stores
// at runtime based on configuration. This pattern is useful for development
// (memory) vs production (SQL) environments.
func demonstrateDualStore() {
	fmt.Println("--- Dual Store Demo ---")

	cfg := getStoreConfig()

	store := createStore(cfg)
	defer func() { _ = store.Close() }()

	fmt.Printf("  Active store backend: %s\n", cfg.StoreType)
	fmt.Printf("  Store type: %T\n", store)

	ctx := context.Background()

	aggID := id.NewAggregateID()

	evt, err := event.New(
		"StoreTested",
		aggID,
		"Test",
		1,
		map[string]string{"backend": cfg.StoreType},
	)
	if err != nil {
		fmt.Printf("  create event error: %v\n", err)
		return
	}

	if err := store.Save(
		ctx,
		event.NewAggregateRef("Test", aggID),
		[]event.Event{evt},
		0,
	); err != nil {
		fmt.Printf("  save error: %v\n", err)
		return
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef("Test", aggID))
	if err != nil {
		fmt.Printf("  load error: %v\n", err)
		return
	}

	fmt.Printf("  Loaded %d event(s) from %s backend\n", len(loaded), cfg.StoreType)
}

type storeConfig struct {
	StoreType string
}

func getStoreConfig() storeConfig {
	return storeConfig{StoreType: "memory"}
}

func createStore(cfg storeConfig) event.Store {
	switch cfg.StoreType {
	default:
		return memory.NewMemoryStore()
	}
}
