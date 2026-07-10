package projectionhost_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

// TestIntegration_ProjectionHost_10KEvents verifies the host processes a large
// journal without dropping events, losing checkpoints, or blocking. This is the
// stress test that the single-event replay-bug-hiding tests failed to be.
func TestIntegration_ProjectionHost_10KEvents(t *testing.T) {
	t.Parallel()

	const eventCount = 10_000

	store := memory.NewMemoryStore()
	defer store.Close()
	cpStore := memory.NewMemoryCheckpointStore()

	host, err := projectionhost.New(
		store, cpStore,
		projectionhost.WithBatchSize(500),
		projectionhost.WithMaxRestarts(-1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	counter := &stressCounter{name: "stress"}
	if err := host.Register(counter); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Seed 10K events into the real store.
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Item", aggID)
	for i := range eventCount {
		typ := event.Type(fmt.Sprintf("item.tick.%d", i%10))
		evt, _ := event.New(typ, aggID, "Item", 1, []byte("payload"))
		if err := store.AppendBatch(context.Background(), ref, []event.Event{evt}); err != nil {
			t.Fatalf("AppendBatch %d: %v", i, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := host.Start(ctx); err != nil {
			t.Errorf("Start: %v", err)
		}
	}()

	requireEventually(t, 30*time.Second, func() bool {
		return counter.count.Load() == int64(eventCount)
	})
	cancel()
	_ = host.Stop()

	if got := counter.count.Load(); got != int64(eventCount) {
		t.Fatalf("expected %d processed, got %d", eventCount, got)
	}

	cp, err := cpStore.Load(context.Background(), "stress")
	if err != nil {
		t.Fatalf("checkpoint Load: %v", err)
	}
	if cp.IsZero() {
		t.Fatal("expected non-zero checkpoint after 10K events")
	}
}

type stressCounter struct {
	name  string
	count atomic.Int64
}

func (p *stressCounter) Name() string { return p.name }

func (p *stressCounter) EventTypes() []event.Type { return nil }

func (p *stressCounter) Handle(_ context.Context, _ event.Event) error {
	p.count.Add(1)

	return nil
}
