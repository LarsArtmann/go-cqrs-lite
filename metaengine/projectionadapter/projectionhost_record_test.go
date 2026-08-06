package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestProjectionHost_RecordAwareLifecycle verifies that Record metadata
// (StreamID, Version) flows through the full projectionhost lifecycle:
// Host.Start → journal replay → adapter.Handle → ApplyRecord → OnRecord fold.
// This is the integration-level proof that the ES-native Record pipeline works
// end-to-end, not just in unit tests.
func TestProjectionHost_RecordAwareLifecycle(t *testing.T) {
	t.Parallel()

	type recordItem struct {
		ID       string
		StreamID string
		Version  int64
		CorrID   string
	}

	type recordQuery struct{}

	type itemEvent struct {
		ID string
	}

	corrID := id.NewCorrelationID()

	q := metaengine.Query[recordQuery, recordItem](
		"record-items",
		metaengine.OnRecord(itemEvent{}, func(rec record.Record, e itemEvent) (string, recordItem) {
			return e.ID, recordItem{
				ID:       e.ID,
				StreamID: rec.StreamID.String(),
				Version:  rec.Version,
				CorrID:   rec.MetaData.CorrelationID,
			}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	decoder := func(eventType string, payload []byte) (any, error) {
		var e itemEvent
		return e, json.Unmarshal(payload, &e)
	}

	adapter := projectionadapter.New("record-items", store, decoder)

	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		"itemEvent", streamID, "Item", event.Version(5),
		mustJSON(t, itemEvent{ID: "item-1"}),
		event.WithCorrelationID(corrID),
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	journal := &memoryJournal{}
	journal.append(evt)

	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(adapter); err != nil {
		t.Fatalf("host.Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = host.Start(ctx) }()
	defer func() { _ = host.Stop() }()

	waitForProcessed(t, host, "record-items", 1)

	result, err := store.Execute(recordQuery{})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	items, ok := result.(map[string]recordItem)
	if !ok {
		t.Fatalf("expected map[string]recordItem, got %T", result)
	}

	item, exists := items["item-1"]
	if !exists {
		t.Fatal("item-1 not found in store")
	}

	if item.StreamID == "" {
		t.Error("StreamID is empty — Record metadata not flowing through Host lifecycle")
	}

	if item.Version != 5 {
		t.Errorf("Version: got %d, want 5", item.Version)
	}

	if item.CorrID != corrID.String() {
		t.Errorf("CorrelationID: got %q, want %q", item.CorrID, corrID.String())
	}
}

// TestProjectionHost_CheckpointAdvances verifies the checkpoint advances
// after processing events through the Host lifecycle.
func TestProjectionHost_CheckpointAdvances(t *testing.T) {
	t.Parallel()

	type cpQuery struct{}
	type cpEvent struct{ ID string }
	type cpResult struct{ ID string }

	q := metaengine.Query[cpQuery, cpResult](
		"cp-items",
		metaengine.On(cpEvent{}, func(e cpEvent) (string, cpResult) {
			return e.ID, cpResult{ID: e.ID}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	decoder := func(eventType string, payload []byte) (any, error) {
		var e cpEvent
		return e, json.Unmarshal(payload, &e)
	}

	adapter := projectionadapter.New("cp-items", store, decoder)

	streamID := id.NewStreamID()
	evt1, _ := event.NewEvent("cpEvent", streamID, "Item", event.Version(1),
		mustJSON(t, cpEvent{ID: "a"}))
	evt2, _ := event.NewEvent("cpEvent", streamID, "Item", event.Version(2),
		mustJSON(t, cpEvent{ID: "b"}))

	journal := &memoryJournal{}
	journal.append(evt1)
	journal.append(evt2)

	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(adapter); err != nil {
		t.Fatalf("host.Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = host.Start(ctx) }()
	defer func() { _ = host.Stop() }()

	waitForProcessed(t, host, "cp-items", 2)

	cp, err := cpStore.Load(context.Background(), "cp-items")
	if err != nil {
		t.Fatalf("checkpoint Load: %v", err)
	}

	if cp.EventID.IsZero() {
		t.Error("checkpoint EventID is zero — checkpoint not advancing")
	}
}
