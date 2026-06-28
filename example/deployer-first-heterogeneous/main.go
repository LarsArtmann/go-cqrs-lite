// Command deployer-first-heterogeneous demonstrates mixing storage engines
// per concern: Pebble for the event store (fast LSM writes) and SQLite for
// materialized views (queryable SQL). This is the deployer-first "killer
// feature" — the deployer picks the right engine for each access pattern,
// and the consumer code doesn't know or care which engine is which.
//
// Run: go run .
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGo

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	pebblestore "github.com/larsartmann/go-cqrs-lite/storage/pebble/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, _ := os.MkdirTemp("", "cqrs-heterogeneous-*")
	defer os.RemoveAll(dir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// ── DEPLOYER: heterogeneous engine mixing ──────────────────────────
	//
	// Pebble for events: LSM-tree optimised for heavy sequential appends.
	// SQLite for views: B-tree with SQL query engine for flexible reads.
	//
	// This is NOT multi-DB split (same engine, different files).
	// This is multi-ENGINE: fundamentally different storage engines,
	// each chosen for its access pattern.

	// 1. Open Pebble for the event-sourcing write model
	pebbleBackend, err := pebblestore.Open(dir+"/pebble", pebblestore.DefaultOptions(), logger)
	if err != nil {
		panic(err)
	}
	defer pebbleBackend.Close()

	// 2. Open SQLite for the read model (materialized views)
	sqlDB, err := storage.OpenSQLite(dir + "/views.db")
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	if err := storage.SQLiteInitSchema(ctx, sqlDB); err != nil {
		panic(err)
	}

	sqlKV, err := storage.NewSQLiteKVStore(sqlDB)
	if err != nil {
		panic(err)
	}

	// 3. Mix them in one Bundle — consumer code is engine-agnostic
	bundle, err := stack.New(
		stack.WithEventStore(pebbleBackend.EventStore()),
		stack.WithSnapshotStore(pebbleBackend.SnapshotStore()),
		stack.WithCheckpointStore(pebbleBackend.CheckpointStore()),
		stack.WithReadModels(sqlKV), // SQL for views, Pebble for events
	)
	if err != nil {
		panic(err)
	}
	defer bundle.Close()

	// ── CONSUMER: engine-agnostic code ──────────────────────────────────
	//
	// The consumer doesn't know events are in Pebble and views are in SQLite.
	// It just uses the Bundle's capabilities.

	// Save an event — goes to Pebble
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("counter.incremented", aggID, "Counter", 1, []byte(`{"by":1}`))
	if err != nil {
		panic(err)
	}

	ref := event.NewAggregateRef("Counter", aggID)
	if err := bundle.EventSink.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		panic(err)
	}

	fmt.Printf("Event persisted to Pebble: %s v%d\n", evt.Type(), evt.Version())

	// Write and query a read model — goes to SQLite
	viewStore := kv.NewTypedStore[int, id.AggregateID](sqlKV)
	version := int(evt.Version())
	_ = viewStore.Set(ctx, aggID, &version)

	count, _ := viewStore.Get(ctx, aggID)
	fmt.Printf("Read model queried from SQLite: counter at version %d\n", *count)
	fmt.Println("\nHeterogeneous mixing works: Pebble writes + SQLite reads, one Bundle.")
}
