package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// benchItem is a minimal struct for benchmarking the adapter pipeline.
type benchItem struct {
	ID   string
	Name string
}

type benchQuery struct{}

// BenchmarkHandle_ApplyRecord measures the overhead of the ApplyRecord path
// (the current implementation that converts events to Record before applying).
// This establishes a baseline for future optimization work.
func BenchmarkHandle_ApplyRecord(b *testing.B) {
	q := metaengine.Query[benchQuery, benchItem](
		"bench-items",
		metaengine.OnRecord(benchItem{}, func(rec record.Record, e benchItem) (string, benchItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		b.Fatalf("metaengine.Plan: %v", err)
	}

	decoder := func(eventType string, payload []byte) (any, error) {
		var e benchItem
		return e, json.Unmarshal(payload, &e)
	}

	adapter := projectionadapter.New("bench-items", store, decoder)

	streamID := id.NewStreamID()
	payload, _ := json.Marshal(benchItem{ID: "bench-1", Name: "test"})
	evt, err := event.NewEvent(
		"benchItem", streamID, "Item", event.Version(1),
		payload,
	)
	if err != nil {
		b.Fatalf("event.NewEvent: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		if err := adapter.Handle(ctx, evt); err != nil {
			b.Fatalf("Handle: %v", err)
		}
	}
}

// BenchmarkHandle_AutoInsert benchmarks the auto-fold path with Record stamping.
func BenchmarkHandle_AutoInsert(b *testing.B) {
	type autoEvent struct {
		ID   string
		Name string
	}
	type autoResult struct {
		ID       string
		Name     string
		StreamID string
		Version  int64
	}

	q := metaengine.Query[benchQuery, autoResult](
		"auto-items",
		metaengine.AutoInsert[autoEvent, autoResult]("ID"),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		b.Fatalf("metaengine.Plan: %v", err)
	}

	decoder := func(eventType string, payload []byte) (any, error) {
		var e autoEvent
		return e, json.Unmarshal(payload, &e)
	}

	adapter := projectionadapter.New("auto-items", store, decoder)

	streamID := id.NewStreamID()
	payload, _ := json.Marshal(autoEvent{ID: "auto-1", Name: "test"})
	evt, err := event.NewEvent(
		"autoEvent", streamID, "Item", event.Version(1),
		payload,
	)
	if err != nil {
		b.Fatalf("event.NewEvent: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		if err := adapter.Handle(ctx, evt); err != nil {
			b.Fatalf("Handle: %v", err)
		}
	}
}
