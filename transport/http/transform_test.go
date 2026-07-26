package http

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestCBORToJSONTransform_CBOR_Event verifies the ready-made transform decodes
// a CBOR-stamped event payload and re-emits it as JSON.
func TestCBORToJSONTransform_CBOR_Event(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	typed := struct {
		Name string `cbor:"name"`
	}{Name: "alice"}

	evt, err := event.New("user.created", streamID, "User", 1, typed)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("encoding = %q, want cbor", evt.Encoding())
	}

	out := CBORToJSONTransform(evt)

	if !strings.Contains(string(out), `"name":"alice"`) {
		t.Errorf("expected JSON with name field; got %q", out)
	}
}

// TestCBORToJSONTransform_JSON_Passthrough verifies that non-CBOR (JSON)
// events pass through unchanged — zero transcoding overhead for JSON stores.
func TestCBORToJSONTransform_JSON_Passthrough(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	typed := struct {
		Name string `json:"name"`
	}{Name: "bob"}

	evt, err := event.New("user.created", streamID, "User", 1, typed,
		event.WithCodec(codec.JSONCodec{}))
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingJSON {
		t.Fatalf("encoding = %q, want json", evt.Encoding())
	}

	out := CBORToJSONTransform(evt)

	if !strings.Contains(string(out), `"name":"bob"`) {
		t.Errorf("JSON payload should pass through; got %q", out)
	}
}

// TestSSEHandler_CBORToJSONTransform_Wire verifies the end-to-end browser flow:
// a typed CBOR-stamped event is published to the bus, and the ready-made
// CBORToJSONTransform (passed to WithPayloadTransform) produces valid JSON on
// the SSE wire. This is the one-liner consumer path that deletes per-consumer
// transcode logic.
func TestSSEHandler_CBORToJSONTransform_Wire(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus, WithPayloadTransform(CBORToJSONTransform))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	rec, stop := startSSE(broker, "cbor-ready", "")
	time.Sleep(50 * time.Millisecond)

	streamID := id.NewStreamID()
	typed := struct {
		Name string `cbor:"name"`
	}{Name: "alice-cbor"}

	evt, err := event.New("user.created", streamID, "User", 1, typed)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("expected CBOR encoding, got %q", evt.Encoding())
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	stop()

	body := rec.Body.String()

	if !strings.Contains(body, `{"name":"alice-cbor"}`) {
		t.Errorf("transformed CBOR→JSON payload missing from wire; body: %q", body)
	}
}
