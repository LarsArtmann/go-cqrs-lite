// Command deployer-first-multidb demonstrates the MULTI-DATABASE split pattern.
//
// Unlike the deployer-first example (which uses a single in-memory store), this
// example wires a SQLite preset with three separate database files: one for
// events, one for commands/queries, and one for materialized views. The
// consumer code is identical — only the deployer wiring changes.
//
// Run: go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, _ := os.MkdirTemp("", "cqrs-multidb-*")
	defer os.RemoveAll(dir)

	// ── DEPLOYER: multi-DB split ────────────────────────────────────────
	//
	// Each concern gets its own database file. This isolates write-heavy event
	// appends from read-model scans and command/query audit traffic.
	//
	// For a single-database deployment, just use sqlite.New("app.db") without
	// any With*DB options.
	bundle, err := sqlite.New(
		filepath.Join(dir, "primary.db"),
		sqlite.WithEventDB(filepath.Join(dir, "events.db")),
		sqlite.WithQueryDB(filepath.Join(dir, "queries.db")),
		sqlite.WithViewDB(filepath.Join(dir, "views.db")),
	)
	if err != nil {
		log.Fatalf("deployer: sqlite.New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// ── CONSUMER: domain logic (identical to single-DB) ────────────────
	mat, err := stack.NewMaterialize[CounterView, id.AggregateID](
		bundle,
		codec.JSONCodec{},
		func(evt event.Event) (id.AggregateID, error) {
			return evt.AggregateID(), nil
		},
	)
	if err != nil {
		log.Fatalf("consumer: NewMaterialize: %v", err)
	}

	configureMaterialize(mat)

	repo, err := stack.Repository[CounterState](bundle, decider.Decider[CounterState]{
		Initial: CounterState{},
		Apply:   applyCounter,
	})
	if err != nil {
		log.Fatalf("consumer: Repository: %v", err)
	}

	counterID := id.NewAggregateID()

	for range 3 {
		if err := repo.Execute(ctx, counterID, "Counter", decideIncrement(counterID)); err != nil {
			log.Fatalf("increment: %v", err)
		}
	}

	// ── Projection: replay journal into materialized view ──────────────
	catchUp, err := bundle.CatchUpSubscriber()
	if err != nil {
		log.Fatalf("consumer: CatchUpSubscriber: %v", err)
	}

	msgs, err := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)
	if err != nil {
		log.Fatalf("consumer: Subscribe: %v", err)
	}

	handler := mat.HandlerFunc()

	go func() {
		for msg := range msgs {
			_ = handler(msg)
			msg.Ack()
		}
	}()

	view, ok := waitForView(ctx, mat, counterID, func(v *CounterView) bool {
		return v.Count == 3
	})
	if !ok {
		log.Fatal("timed out waiting for materialized view")
	}

	fmt.Fprintf(os.Stderr, "counter %s: count=%d (events in events.db, view in views.db)\n",
		counterID, view.Count)
}
