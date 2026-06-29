// Package main demonstrates the managed projection host (projectionhost/v3).
//
// It shows the full reliability story in one runnable program:
//   - A real event store (storage/memory) feeds the host's SeekableJournal.
//   - Two projections run concurrently, each in its own goroutine.
//   - A poison event (a projection that fails) is captured by the dead-letter
//     queue, the checkpoint advances, and the stream keeps flowing.
//   - After "shipping a fix", ReplayDeadLetters re-feeds the poisoned event
//     and purges it on success.
//
// Run with: go run ./example/projectionhost
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

func main() {
	ctx := context.Background()

	// --- Infrastructure (a consumer would use a real store in production) ---
	store := memory.NewMemoryStore()
	defer store.Close()

	cpStore := memory.NewMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	// --- A read-model projection that tracks a counter ---
	counter := &counterProjection{name: "counter"}

	// --- A projection with a transient bug (fails on "order.cancelled") ---
	buggy := &buggyProjection{name: "orders", poisonType: "order.cancelled"}

	host, err := projectionhost.New(
		store, cpStore,
		projectionhost.WithBatchSize(10),
		projectionhost.WithDeadLetterStore(dlq, 2), // poison after 2 retries
		projectionhost.WithMaxRestarts(-1),         // keep restarting for the demo
		projectionhost.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		}))),
	)
	if err != nil {
		panic(err)
	}

	for _, p := range []projection.Projection{counter, buggy} {
		if err := host.Register(p); err != nil {
			panic(err)
		}
	}

	// --- Seed events BEFORE starting the host (the host is a catch-up
	// processor: it drains the journal then stops; for live push delivery use
	// watermill/CatchUpSubscriber alongside the host) ---
	ref := event.NewAggregateRef("Order", id.NewAggregateID())
	for _, typ := range []event.Type{"order.created", "order.created", "order.created"} {
		evt, _ := event.New(typ, ref.ID, "Order", 1, []byte("payload"))
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	poison, _ := event.New("order.cancelled", ref.ID, "Order", 1, []byte("payload"))
	_ = store.AppendBatch(ctx, ref, []event.Event{poison})

	// --- Start the host (processes the pre-seeded journal, then idles) ---
	runCtx, cancel := context.WithCancel(ctx)
	if err := host.Start(runCtx); err != nil {
		panic(err)
	}

	// --- Wait for the poison event to land in the DLQ ---
	waitFor(2*time.Second, func() bool {
		entries, _ := dlq.List(ctx, "")

		return len(entries) == 1
	})

	cancel()

	_ = host.Stop()

	fmt.Println("\n--- Status ---")

	for _, s := range host.Status() {
		fmt.Printf("  %s: status=%s processed=%d errors=%d\n",
			s.Name, s.Status, s.Processed, s.Errors)
	}

	entries, _ := dlq.List(ctx, "")
	fmt.Printf("\n--- Dead-Letter Queue (%d entry) ---\n", len(entries))

	for _, e := range entries {
		fmt.Printf("  %s: event=%s error=%q\n", e.ProjectionName, e.EventType, e.Error)
	}

	// --- "We shipped a fix": replay the DLQ with a corrected projection ---
	buggy.fixed = true // simulate the handler fix

	result, err := host.ReplayDeadLetters(ctx, "")
	if err != nil {
		panic(err)
	}
	// ReplayDeadLetters is pure — caller decides whether to purge the successes.
	_ = dlq.Purge(ctx, "orders")

	remaining, _ := dlq.List(ctx, "")
	fmt.Printf("\n--- Replay: %d succeeded, %d still failing, %d remain in DLQ ---\n",
		len(result.Replayed), len(result.StillFailing), len(remaining))
}

// counterProjection counts every event it sees.
type counterProjection struct {
	name  string
	mu    sync.Mutex
	count int
}

func (p *counterProjection) Name() string { return p.name }

func (p *counterProjection) EventTypes() []event.Type { return nil } // all types

func (p *counterProjection) Handle(_ context.Context, evt event.Event) error {
	p.mu.Lock()
	p.count++
	p.mu.Unlock()

	return nil
}

// buggyProjection fails on a specific event type until `fixed` is toggled,
// simulating a handler bug that is later corrected.
type buggyProjection struct {
	name       string
	poisonType event.Type
	mu         sync.Mutex
	fixed      bool
}

func (p *buggyProjection) Name() string { return p.name }
func (p *buggyProjection) EventTypes() []event.Type {
	return []event.Type{"order.created", "order.cancelled"}
}

func (p *buggyProjection) Handle(_ context.Context, evt event.Event) error {
	p.mu.Lock()
	fixed := p.fixed
	p.mu.Unlock()

	if !fixed && evt.Type() == p.poisonType {
		return errors.New("handler bug: cannot process cancellation yet")
	}

	return nil
}

func waitFor(timeout time.Duration, cond func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}
}
