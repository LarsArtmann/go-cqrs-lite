package http

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// TestBackfillHandler_CBORToJSONTransform verifies the ready-made adapter works
// through the REST backfill path: a CBOR-stamped event in the journal is served
// as transcoded JSON by BackfillHandler, which reuses the broker's transform.
// This proves the same one-liner configuration covers both SSE streaming and
// REST backfill — no separate transform is needed per delivery path.
func TestBackfillHandler_CBORToJSONTransform(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)

	// evt0 establishes the "after" cursor; evt1 is the CBOR event we backfill.
	evt0, _ := event.NewEvent("test.event", streamID, "Test", 1, []byte(`{"seq":0}`))
	typed := struct {
		Name string `cbor:"name"`
	}{Name: "backfill-cbor"}
	evt1, err := event.New("test.event", streamID, "Test", 2, typed)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	if evt1.Encoding() != codec.EncodingCBOR {
		t.Fatalf("expected CBOR encoding, got %q", evt1.Encoding())
	}

	_ = store.Save(context.Background(), ref, []event.Event{evt0, evt1}, 0)

	bus := eventtest.NewFakeBus()
	defer bus.Close()
	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 100),
		WithPayloadTransform(CBORToJSONTransform),
	)
	if err != nil {
		t.Fatalf("NewSSEBroker: %v", err)
	}
	defer broker.Close()

	handler := BackfillHandler(broker)
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/backfill?after="+evt0.ID().String()+"&limit=10",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, `{"name":"backfill-cbor"}`) {
		t.Errorf("CBOR→JSON transcoded payload missing from backfill response; body: %q", body)
	}
}

// corruptCBORCodec stamps EncodingCBOR but emits invalid CBOR bytes, to
// exercise CBORToJSONTransform's graceful-fallback path without depending on
// event internals.
type corruptCBORCodec struct{}

func (corruptCBORCodec) Encoding() codec.Encoding   { return codec.EncodingCBOR }
func (corruptCBORCodec) Encode(any) ([]byte, error) { return []byte{0xa1, 0xff, 0xff}, nil }
func (corruptCBORCodec) Decode([]byte, any) error   { return nil }

// TestCBORToJSONTransform_CorruptCBOR_FallsBackToRaw verifies that when the
// payload is stamped CBOR but cannot be decoded, the transform returns the raw
// payload unchanged — graceful degradation so SSE clients always receive data
// rather than a gap (UP1 acceptance: transform error → raw payload sent).
func TestCBORToJSONTransform_CorruptCBOR_FallsBackToRaw(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt, err := event.New("test.event", streamID, "Test", 1,
		map[string]any{"x": 1}, event.WithCodec(corruptCBORCodec{}))
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("encoding = %q, want cbor", evt.Encoding())
	}

	raw := event.PayloadReadOnly(evt)
	out := CBORToJSONTransform(evt)

	if string(out) != string(raw) {
		t.Errorf("expected graceful fallback to raw payload; got %q, want %q", out, raw)
	}
}
