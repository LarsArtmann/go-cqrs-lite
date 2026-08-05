package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── Test types for TypeDecoder ───

type createdPayload struct {
	Title string `json:"title"`
}

type assignedPayload struct {
	Assignee string `json:"assignee"`
}

type deletedPayload struct{}

type statsInput struct{}

// ─── Register + TypeDecoder tests ───

func TestTypeDecoder_Register_DecodesPayload(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), createdPayload{}),
		projectionadapter.Register(event.Type("item.assigned"), assignedPayload{}),
		projectionadapter.Register(event.Type("item.deleted"), deletedPayload{}),
	)

	evt := makeTypedEvent(t, "item.created", createdPayload{Title: "Hello"})

	got, err := dec.Decode(evt)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	wrapped, ok := got.(projectionadapter.EventWithID[createdPayload])
	if !ok {
		t.Fatalf("got %T, want EventWithID[createdPayload]", got)
	}

	if wrapped.Payload.Title != "Hello" {
		t.Fatalf("Title = %q, want %q", wrapped.Payload.Title, "Hello")
	}

	if wrapped.ID == "" {
		t.Fatal("ID should be the stream ID, got empty string")
	}

	if wrapped.ID != evt.StreamID().String() {
		t.Fatalf("ID = %q, want %q", wrapped.ID, evt.StreamID().String())
	}
}

func TestTypeDecoder_RegisterString_DecodesPayload(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.RegisterString("item.created", createdPayload{}),
	)

	evt := makeTypedEvent(t, "item.created", createdPayload{Title: "World"})

	got, err := dec.Decode(evt)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	wrapped, ok := got.(projectionadapter.EventWithID[createdPayload])
	if !ok {
		t.Fatalf("got %T, want EventWithID[createdPayload]", got)
	}

	if wrapped.Payload.Title != "World" {
		t.Fatalf("Title = %q, want %q", wrapped.Payload.Title, "World")
	}
}

func TestTypeDecoder_UnregisteredEventType_ReturnsError(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), createdPayload{}),
	)

	evt := makeTypedEvent(t, "item.unknown", createdPayload{})

	_, err := dec.Decode(evt)
	if !errors.Is(err, projectionadapter.ErrNoFoldForEventType) {
		t.Fatalf("err = %v, want ErrNoFoldForEventType", err)
	}
}

func TestTypeDecoder_EmptyPayload_HandledGracefully(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.deleted"), deletedPayload{}),
	)

	// Create an event with empty payload.
	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		event.Type("item.deleted"), streamID, "Item", event.Version(1),
		[]byte{},
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	got, err := dec.Decode(evt)
	if err != nil {
		t.Fatalf("Decode with empty payload: %v", err)
	}

	wrapped, ok := got.(projectionadapter.EventWithID[deletedPayload])
	if !ok {
		t.Fatalf("got %T, want EventWithID[deletedPayload]", got)
	}

	// Zero-value payload is fine.
	_ = wrapped
}

func TestTypeDecoder_EventTypes_ReturnsAllRegistered(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), createdPayload{}),
		projectionadapter.Register(event.Type("item.deleted"), deletedPayload{}),
	)

	types := dec.EventTypes()
	if len(types) != 2 {
		t.Fatalf("EventTypes() returned %d types, want 2", len(types))
	}
}

func TestTypeDecoder_InvalidJSON_ReturnsDecodeError(t *testing.T) {
	t.Parallel()

	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), createdPayload{}),
	)

	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		event.Type("item.created"), streamID, "Item", event.Version(1),
		[]byte("not valid json"),
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	_, err = dec.Decode(evt)
	if err == nil {
		t.Fatal("Decode should fail on invalid JSON")
	}
}

// ─── NewWithDecoder integration tests ───

func TestNewWithDecoder_EndToEnd(t *testing.T) {
	t.Parallel()

	// Declare a Counter query keyed by stream ID.
	q := metaengine.Query[statsInput, map[string]int64](
		"item_counts",
		metaengine.OnTyped(
			"item.created",
			projectionadapter.EventWithID[createdPayload]{},
			func(_ projectionadapter.EventWithID[createdPayload]) metaengine.Delta {
				return metaengine.Delta{"created": 1}
			},
		),
		metaengine.OnTyped(
			"item.deleted",
			projectionadapter.EventWithID[deletedPayload]{},
			func(_ projectionadapter.EventWithID[deletedPayload]) metaengine.Delta {
				return metaengine.Delta{"created": -1}
			},
		),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	defer store.Close()

	// Build the decoder the new way (replaces the 77-line switch).
	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), createdPayload{}),
		projectionadapter.Register(event.Type("item.deleted"), deletedPayload{}),
	)

	adapter := projectionadapter.NewWithDecoder("items", store, dec)

	// Handle events.
	evt1 := makeTypedEvent(t, "item.created", createdPayload{Title: "First"})
	evt2 := makeTypedEvent(t, "item.created", createdPayload{Title: "Second"})

	if err := adapter.Handle(context.Background(), evt1); err != nil {
		t.Fatalf("Handle evt1: %v", err)
	}

	if err := adapter.Handle(context.Background(), evt2); err != nil {
		t.Fatalf("Handle evt2: %v", err)
	}

	// Verify the counter.
	result, err := store.Execute(statsInput{})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	counts, ok := result.(map[string]int64)
	if !ok {
		t.Fatalf("result is %T, want map[string]int64", result)
	}

	if counts["created"] != 2 {
		t.Fatalf("counts[created] = %d, want 2", counts["created"])
	}
}

// ─── Helpers ───

func makeTypedEvent(t *testing.T, eventType string, payload any) event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	evt, err := event.NewEvent(
		event.Type(eventType), streamID, "Item", event.Version(1),
		payloadBytes,
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	return evt
}
