package projection_test

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}

var update = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return filepath.Join("testdata", "golden")
}

type processedEvent struct {
	EventType string `json:"eventType"`
	AggID     string `json:"aggregateId"`
	Version   int    `json:"version"`
}

type trackedHandler struct {
	mu        sync.Mutex
	name      string
	types     []event.Type
	processed []processedEvent
}

func (h *trackedHandler) Name() string             { return h.name }
func (h *trackedHandler) EventTypes() []event.Type { return h.types }

func (h *trackedHandler) Handle(_ context.Context, evt event.Event) error {
	h.mu.Lock()
	h.processed = append(h.processed, processedEvent{
		EventType: string(evt.Type()),
		AggID:     evt.AggregateID().String(),
		Version:   int(evt.Version()),
	})
	h.mu.Unlock()
	return nil
}

func TestGolden_ReplayOrder(t *testing.T) {
	cat := buildReplayCatalog(t)

	got, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "replay-order.json")
	eventtest.AssertGolden(t, goldenPath, got, *update)
}

func buildReplayCatalog(t *testing.T) []processedEvent {
	t.Helper()

	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	aggID := parseAggID("01ARYZ6S41TSV4RRFFQ69G5FAV")
	ref := event.NewAggregateRef("Order", aggID)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	types := []event.Type{"OrderCreated", "ItemAdded", "ItemAdded", "OrderShipped"}

	evts := make([]event.Event, 0, len(types))

	for i, typ := range types {
		evt, err := event.New(typ, aggID, "Order", event.Version(i+1),
			map[string]any{"index": i}, event.WithOccurredAt(now.Add(time.Duration(i)*time.Minute)))
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		evts = append(evts, evt)
	}

	ctx := context.Background()

	if err := store.Save(ctx, ref, evts, event.Version(0)); err != nil {
		t.Fatalf("save: %v", err)
	}

	bus := eventtest.NewFakeBus()
	t.Cleanup(func() { _ = bus.Close() })

	cp := memory.NewMemoryCheckpointStore()
	t.Cleanup(func() { _ = cp.Close() })

	handler := &trackedHandler{name: "order-tracker", types: nil}

	runner, err := projection.NewRunner(store, bus, cp, projection.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}

	if err := runner.Register(handler); err != nil {
		t.Fatalf("register: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan error, 1)

	go func() { done <- runner.Run(runCtx) }()

	waitForHandler(t, handler, len(types), 2*time.Second)

	cancel()
	<-done

	return handler.items()
}

func (h *trackedHandler) len() int {
	h.mu.Lock()
	n := len(h.processed)
	h.mu.Unlock()
	return n
}

func (h *trackedHandler) items() []processedEvent {
	h.mu.Lock()
	out := slices.Clone(h.processed)
	h.mu.Unlock()
	return out
}

func waitForHandler(t *testing.T, h *trackedHandler, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if h.len() >= expected {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for handler: got %d events, want %d", h.len(), expected)
}

func discardLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}),
	)
}
