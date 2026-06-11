package pebble

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestSerializeEvent_CBORFormat(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("UserCreated", aggID, "User", event.Version(1),
		[]byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}

	if !isCBOR(data) {
		t.Errorf("expected CBOR major type 5 (0xa0-0xbf), got first byte 0x%02x", data[0])
	}
}

func TestDeserializeEvent_JSONBackwardCompat(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()

	legacy := serializableEvent{
		ID:            eventID,
		Type:          "UserCreated",
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       1,
		SchemaVersion: 0,
		Payload:       []byte(`{"name":"Bob"}`),
		OccurredAt:    time.Now().UnixNano(),
		Encoding:      "json",
	}

	jsonData, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy JSON: %v", err)
	}

	if isCBOR(jsonData) {
		t.Fatal("JSON data should not be detected as CBOR")
	}

	evt, err := store.deserializeEvent(jsonData)
	if err != nil {
		t.Fatalf("deserialize JSON envelope: %v", err)
	}

	if evt.Type() != "UserCreated" {
		t.Errorf("expected type UserCreated, got %s", evt.Type())
	}

	if evt.Version().Int() != 1 {
		t.Errorf("expected version 1, got %d", evt.Version().Int())
	}

	if evt.AggregateID() != aggID {
		t.Error("aggregate ID mismatch")
	}
}

func TestDeserializeEvent_CBORRoundTrip(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("OrderPlaced", aggID, "Order", event.Version(3),
		[]byte(`{"item":"widget","qty":5}`), event.WithSchemaVersion(2))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if !isCBOR(data) {
		t.Fatal("expected CBOR format")
	}

	got, err := store.deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if got.Type() != evt.Type() {
		t.Errorf("type: want %s, got %s", evt.Type(), got.Type())
	}

	if got.Version() != evt.Version() {
		t.Errorf("version: want %d, got %d", evt.Version(), got.Version())
	}

	if got.AggregateID() != evt.AggregateID() {
		t.Error("aggregate ID mismatch")
	}

	if got.SchemaVersion() != evt.SchemaVersion() {
		t.Errorf("schema version: want %d, got %d", evt.SchemaVersion(), got.SchemaVersion())
	}
}

func TestSerializeEvent_NoBase64(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	payload := make([]byte, 256)
	for i := range payload {
		payload[i] = byte(i)
	}

	evt, err := event.NewEvent("BinaryData", aggID, "Blob", event.Version(1), payload)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if bytes.Contains(data, []byte("base64")) {
		t.Error("found 'base64' in CBOR output — payload should be raw bytes, not base64-encoded")
	}

	got, err := store.deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if !bytes.Equal(event.PayloadReadOnly(got), payload) {
		t.Error("payload mismatch after round-trip")
	}
}

func TestIsCBOR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", nil, false},
		{"json_object", []byte(`{"id":"test"}`), false},
		{"json_array", []byte(`[1,2,3]`), false},
		{"cbor_map_0", []byte{0xa0}, true},
		{"cbor_map_1", []byte{0xa1, 0x01}, true},
		{"cbor_map_23", []byte{0xbf}, true},
		{"cbor_uint", []byte{0x01}, false},
		{"cbor_string", []byte{0x61, 0x41}, false},
		{"cbor_array", []byte{0x80}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isCBOR(tc.data); got != tc.want {
				t.Errorf("isCBOR(%v) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}

func TestSerializeEvent_Deterministic(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("TestEvent", aggID, "Test", event.Version(1),
		[]byte(`{"z":1,"a":2,"m":3}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	var prev []byte

	for i := range 10 {
		data, err := store.serializeEvent(evt)
		if err != nil {
			t.Fatalf("serialize %d: %v", i, err)
		}

		if prev != nil && !bytes.Equal(prev, data) {
			t.Fatal("serialize is not deterministic — same event produced different bytes")
		}

		prev = data
	}
}

func TestEventStore_Persistence_CBOR(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	db, err := pebble.Open( //nolint:varnamelen
		dir,
		&pebble.Options{},
	)
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	aggID := id.NewAggregateID()
	ctx := context.Background()

	store := NewStore(db, nil)

	evt, err := event.NewEvent("ItemCreated", aggID, "Item", event.Version(1),
		[]byte(`{"sku":"W-001"}`), event.WithCorrelationID(id.NewCorrelationID()))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	ref := event.NewAggregateRef("Item", aggID)

	err = store.Save(ctx, ref, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("reopen pebble: %v", err)
	}

	t.Cleanup(func() { _ = db2.Close() })

	store2 := NewStore(db2, nil)

	loaded, err := store2.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load after reopen: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	got := loaded[0]
	if got.Type() != "ItemCreated" {
		t.Errorf("type: want ItemCreated, got %s", got.Type())
	}

	if got.Version().Int() != 1 {
		t.Errorf("version: want 1, got %d", got.Version().Int())
	}

	if got.Metadata().CorrelationID != evt.Metadata().CorrelationID {
		t.Error("correlation ID mismatch")
	}
}

func TestEventStore_BinaryPayload(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()
	ctx := context.Background()

	payload := make([]byte, 512)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	evt, err := event.NewEvent("BinaryEvent", aggID, "Bin", event.Version(1), payload)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	ref := event.NewAggregateRef("Bin", aggID)

	err = store.Save(ctx, ref, []event.Event{evt}, event.Version(0))
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if !bytes.Equal(event.PayloadReadOnly(loaded[0]), payload) {
		t.Error("binary payload corrupted after save/load round-trip")
	}
}

func TestDeserializeEvent_EmptyData(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)

	_, err := store.deserializeEvent(nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}

	_, err = store.deserializeEvent([]byte{})
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestSerializeEvent_SmallerThanJSON(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()

	payload := make([]byte, 100)
	for i := range payload {
		payload[i] = byte(i)
	}

	evt, err := event.NewEvent("BinaryEvent", aggID, "Bin", event.Version(1), payload)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	s := serializableEvent{
		ID:            evt.ID(),
		Type:          string(evt.Type()),
		AggregateID:   evt.AggregateID(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		Payload:       payload,
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Encoding:      string(evt.Encoding()),
	}

	cborData, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serialize CBOR: %v", err)
	}

	jsonData, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}

	if len(cborData) >= len(jsonData) {
		t.Errorf("CBOR (%d bytes) should be smaller than JSON (%d bytes) for binary payload",
			len(cborData), len(jsonData))
	}
}

func TestDeserializeEvent_CBORWithMetadata(t *testing.T) {
	t.Parallel()

	store := newPebbleTestStore(t)
	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()
	causeID := id.NewCausationID()

	evt, err := event.NewEvent("WithMeta", aggID, "Test", event.Version(1),
		[]byte(`{"x":1}`),
		event.WithCorrelationID(corrID),
		event.WithCausationID(causeID))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	got, err := store.deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if got.Metadata().CorrelationID != corrID {
		t.Errorf("correlation ID: want %s, got %s", corrID, got.Metadata().CorrelationID)
	}

	if got.Metadata().CausationID != causeID {
		t.Errorf("causation ID: want %s, got %s", causeID, got.Metadata().CausationID)
	}
}
