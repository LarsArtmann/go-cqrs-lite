package projectionhost_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// integrationProjection is a real read-model projection that counts
// successfully handled events.
type integrationProjection struct {
	name  string
	count atomic.Int64
}

func (p *integrationProjection) Name() string { return p.name }

func (p *integrationProjection) EventTypes() []event.Type {
	return []event.Type{"item.added", "item.removed"}
}

func (p *integrationProjection) Handle(_ context.Context, evt event.Event) error {
	if evt.Type() == "item.removed" {
		return errors.New("transient handler bug on remove")
	}
	p.count.Add(1)

	return nil
}

// TestIntegration_ProjectionHost_WithMemoryStore verifies the managed host
// composes with a REAL storage/memory.MemoryStore end-to-end: events saved to
// the store are read via SeekableJournal, projected, checkpointed, and a poison
// event is captured by the dead-letter queue.
func TestIntegration_ProjectionHost_WithMemoryStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()
	cpStore := memory.NewMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	proj := &integrationProjection{name: "items"}

	host, err := projectionhost.New(
		store, cpStore,
		projectionhost.WithBatchSize(5),
		projectionhost.WithDeadLetterStore(dlq, 2),
		projectionhost.WithMaxRestarts(-1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Seed the real store with 3 good events + 1 poison event.
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Item", streamID)
	goodEvents := []event.Type{"item.added", "item.added", "item.added"}
	for _, typ := range goodEvents {
		evt, _ := event.New(typ, streamID, "Item", 1, []byte("payload"))
		if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
			t.Fatalf("Save %s: %v", typ, err)
		}
	}
	poisonEvt, _ := event.New("item.removed", streamID, "Item", 1, []byte("payload"))
	if err := store.AppendBatch(context.Background(), ref, []event.Event{poisonEvt}); err != nil {
		t.Fatalf("Save poison: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := host.Start(ctx); err != nil {
			t.Errorf("Start: %v", err)
		}
	}()

	// Wait for the 3 good events to be projected.
	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 3
	})
	// Wait for the poison event to land in the DLQ.
	requireEventually(t, 3*time.Second, func() bool {
		entries, _ := dlq.List(context.Background(), "")

		return len(entries) == 1
	})

	cancel()
	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Verify: 3 items projected, 1 poisoned.
	if proj.count.Load() != 3 {
		t.Fatalf("expected 3 projected items, got %d", proj.count.Load())
	}
	entries, _ := dlq.List(context.Background(), "")
	if len(entries) != 1 {
		t.Fatalf("expected 1 DLQ entry, got %d", len(entries))
	}
	if entries[0].EventType != "item.removed" {
		t.Fatalf("DLQ entry type: got %q, want %q", entries[0].EventType, "item.removed")
	}
	if entries[0].Event == nil {
		t.Fatal("DLQ entry must carry the original Event for replay")
	}

	// Verify checkpoint was persisted to the real checkpoint store.
	cp, err := cpStore.Load(context.Background(), "items")
	if err != nil {
		t.Fatalf("checkpoint Load: %v", err)
	}
	if cp.IsZero() {
		t.Fatal("expected non-zero checkpoint after processing")
	}

	// Replay the DLQ: register a projection that succeeds. ReplayDeadLetters is
	// pure — it returns ReplayResult without mutating the store; the caller
	// purges on success.
	replayHost, _ := projectionhost.New(
		store, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 2),
	)
	_ = replayHost.Register(&countingProjection{name: "items"})
	result, err := replayHost.ReplayDeadLetters(context.Background(), "")
	if err != nil {
		t.Fatalf("ReplayDeadLetters: %v", err)
	}
	if len(result.Replayed) != 1 {
		t.Fatalf("expected 1 replayed entry, got %d", len(result.Replayed))
	}
	// Pure replay must NOT have emptied the DLQ.
	beforePurge, _ := dlq.List(context.Background(), "")
	if len(beforePurge) != 1 {
		t.Fatalf("pure replay must leave the entry in place; got %d", len(beforePurge))
	}
	_ = dlq.Purge(context.Background(), "items")
	remaining, _ := dlq.List(context.Background(), "")
	if len(remaining) != 0 {
		t.Fatalf("DLQ should be empty after explicit purge, got %d", len(remaining))
	}
}

// Compile-time guard: integrationProjection satisfies the Projection interface.
var _ projection.Projection = (*integrationProjection)(nil)
