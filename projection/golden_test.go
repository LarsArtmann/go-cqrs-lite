package projection_test

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

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
	name      string
	types     []event.Type
	processed []processedEvent
}

func (h *trackedHandler) Name() string             { return h.name }
func (h *trackedHandler) EventTypes() []event.Type { return h.types }

func (h *trackedHandler) Handle(_ context.Context, evt event.Event) error {
	h.processed = append(h.processed, processedEvent{
		EventType: string(evt.Type()),
		AggID:     evt.AggregateID().String(),
		Version:   int(evt.Version()),
	})
	return nil
}

func TestGolden_ReplayOrder(t *testing.T) {
	cat := buildReplayCatalog(t)

	got, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "replay-order.json")
	assertGolden(t, goldenPath, got)
}

func buildReplayCatalog(t *testing.T) []processedEvent {
	t.Helper()

	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	aggID := id.MustParseAggregateID("01ARYZ6S41TSV4RRFFQ69G5FAV")
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

	bus := memory.NewMemoryBus()
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

	return handler.processed
}

func waitForHandler(t *testing.T, h *trackedHandler, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if len(h.processed) >= expected {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for handler: got %d events, want %d", len(h.processed), expected)
}

func discardLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}),
	)
}

func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
