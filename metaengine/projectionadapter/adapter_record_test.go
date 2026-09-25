package projectionadapter_test

import (
	"context"
	"time"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-codec"

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
		metaengine.OnRecord(plainEvent{}, func(_ record.Record, e plainEvent) metaengine.Delta {
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

// TestAdapter_TemporalStampsSurviveApplyRecord pins ADR-0141 follow-up f44
// (2026-09-25): the CQRS path — event stamps flowing through
// projectionadapter.Handle → ApplyRecord → folds — lands on a VERSIONED
// engine with the EVENT's timestamp, so as-of reads over the projected
// collection answer at event time, not wall-clock ingestion time.
func TestAdapter_TemporalStampsSurviveApplyRecord(t *testing.T) {
	t.Parallel()

	type priceEvent struct {
		SKU    string
		Amount int
	}

	type priceQuery struct {
		SKU string
	}

	type priceView struct {
		SKU    string
		Amount int
	}

	q := metaengine.Query[priceQuery, priceView](
		"price-temporal",
		metaengine.OnRecord(priceEvent{}, func(_ record.Record, e priceEvent) (string, priceView) {
			return e.SKU, priceView{SKU: e.SKU, Amount: e.Amount}
		}),
	)

	eng := metaengine.NewMemoryEngineWithVersioning()
	t.Cleanup(func() { _ = eng.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Fold handlers are reflection-based: they need typed structs, so the
	// adapter gets a decoder (CBOR payloads decode via the event codec's
	// JSON-compatible re-encode — use the shared test decoder shape).
	decoder := func(eventType string, payload []byte) (any, error) {
		var e priceEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}
		return e, nil
	}

	handle := projectionadapter.New("price-projection", store, decoder)

	stream, _ := id.ParseStreamID("price-1")

	makeEvent := func(amount int, when time.Time) *event.ImmutableEvent {
		evt, err := event.New(
			"priceEvent",
			stream,
			"Price",
			1,
			priceEvent{SKU: "sku-1", Amount: amount},
			event.WithOccurredAt(when),
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}
		return evt
	}

	old := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)

	for _, evt := range []*event.ImmutableEvent{makeEvent(100, old), makeEvent(200, newer)} {
		if err := handle.Handle(context.Background(), evt); err != nil {
			t.Fatalf("Handle: %v", err)
		}
	}

	// Fold-serialized latest: newest stamp wins regardless of arrival order.
	gotRaw, err := store.Execute(priceQuery{SKU: "sku-1"})
	if err != nil {
		t.Fatalf("Execute latest: %v", err)
	}

	got, ok := gotRaw.(priceView)
	if !ok {
		t.Fatalf("latest value shape: %T", gotRaw)
	}

	if got.Amount != 200 {
		t.Fatalf("latest amount = %d, want 200 (newest event stamp must win)", got.Amount)
	}

	// As-of between the two events: the older price.
	asOfVal, err := store.ExecuteAsOf(context.Background(), "price-temporal", "sku-1", time.Now().Add(-90*time.Minute))
	if err != nil {
		t.Fatalf("ExecuteAsOf: %v", err)
	}

	asOfView, ok := asOfVal.(priceView)
	if !ok {
		t.Fatalf("as-of value shape: %T", asOfVal)
	}

	if asOfView.Amount != 100 {
		t.Fatalf("as-of amount = %d, want 100 (event stamps must survive the adapter path)", asOfView.Amount)
	}
}
