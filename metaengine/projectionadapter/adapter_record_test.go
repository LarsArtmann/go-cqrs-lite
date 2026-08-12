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

// TestAdapter_OnRecordFold_ReceivesRealMetadata verifies that when
// projectionadapter.Handle() processes an event, OnRecord folds receive the
// real StreamID, Version, and metadata — not zero values. This is the key
// integration test proving the record/ wiring is live (ADR-0112).
func TestAdapter_OnRecordFold_ReceivesRealMetadata(t *testing.T) {
	t.Parallel()

	type itemEvent struct {
		ID   string
		Name string
	}

	type itemQuery struct {
		ID string
	}

	type itemView struct {
		ID       string
		Name     string
		StreamID string
		Version  int64
		CorrID   string
		ActorID  string
	}

	// capturedRec stores the Record the fold received for later assertion.
	var capturedRec record.Record

	q := metaengine.Query[itemQuery, itemView](
		"item-by-record",
		metaengine.OnRecord(itemEvent{}, func(rec record.Record, e itemEvent) (string, itemView) {
			capturedRec = rec

			return e.ID, itemView{
				ID:       e.ID,
				Name:     e.Name,
				StreamID: rec.StreamID.String(),
				Version:  rec.Version,
				CorrID:   rec.MetaData.CorrelationID,
				ActorID:  rec.MetaData.ActorID,
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
	defer store.Close()

	// Use a plain PayloadDecoder — for OnRecord folds, the Record carries
	// the StreamID so EventWithID wrapping is unnecessary.
	decoder := func(eventType string, payload []byte) (any, error) {
		var e itemEvent
		err := json.Unmarshal(payload, &e)
		return e, err
	}

	adapter := projectionadapter.New("items-rec", store, decoder)

	// Build a real event with metadata.
	streamID := id.NewStreamID()
	correlationID := id.NewCorrelationID()
	userID := id.NewUserID()

	payloadJSON, err := json.Marshal(itemEvent{ID: "it-1", Name: "Widget"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	evt, err := event.NewEvent(
		"itemEvent", streamID, "Item", event.Version(7),
		payloadJSON,
		event.WithCorrelationID(correlationID),
		event.WithUserID(userID),
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify the fold received the real metadata.
	if capturedRec.StreamID == "" {
		t.Fatal("fold received empty StreamID — record wiring is broken")
	}

	wantStreamID := record.NewStreamRef("Item", streamID.String())
	if capturedRec.StreamID != wantStreamID {
		t.Errorf("StreamID = %q, want %q", capturedRec.StreamID, wantStreamID)
	}

	if capturedRec.Version != 7 {
		t.Errorf("Version = %d, want 7", capturedRec.Version)
	}

	if capturedRec.Type != "itemEvent" {
		t.Errorf("Type = %q, want %q", capturedRec.Type, "itemEvent")
	}

	if capturedRec.MetaData.CorrelationID != correlationID.String() {
		t.Errorf("CorrelationID = %q, want %q",
			capturedRec.MetaData.CorrelationID, correlationID.String())
	}

	if capturedRec.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID = %q, want %q",
			capturedRec.MetaData.ActorID, userID.String())
	}

	// Verify the query result reflects the record context.
	result, err := metaengine.ExecuteTyped[itemQuery, itemView](
		context.Background(), store, itemQuery{ID: "it-1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.StreamID != wantStreamID.String() {
		t.Errorf("result StreamID = %q, want %q", result.StreamID, wantStreamID.String())
	}

	if result.Version != 7 {
		t.Errorf("result Version = %d, want 7", result.Version)
	}
}

// TestAdapter_OnRecordFold_LegacyOnStillWorks verifies that non-Record-aware
// folds (created via On, not OnRecord) still work after the ApplyRecord switch.
func TestAdapter_OnRecordFold_LegacyOnStillWorks(t *testing.T) {
	t.Parallel()

	type plainEvent struct {
		ID    string
		Count int64
	}

	q := metaengine.Query[struct{}, map[string]int64](
		"plain-count",
		metaengine.On(plainEvent{}, func(e plainEvent) metaengine.Delta {
			return metaengine.Delta{e.ID: e.Count}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	decoder := func(eventType string, payload []byte) (any, error) {
		var e plainEvent
		err := json.Unmarshal(payload, &e)
		return e, err
	}

	adapter := projectionadapter.New("plain", store, decoder)

	payload, _ := json.Marshal(plainEvent{ID: "a", Count: 5})
	evt, err := event.NewEvent("plainEvent", id.NewStreamID(), "Item", event.Version(1), payload)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	result, err := metaengine.ExecuteTyped[struct{}, map[string]int64](
		context.Background(), store, struct{}{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result["a"] != 5 {
		t.Errorf("count = %d, want 5", result["a"])
	}
}
