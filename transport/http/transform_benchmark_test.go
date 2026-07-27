package http

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// BenchmarkCBORToJSONTransform_SSEWire measures the end-to-end SSE payload
// transform: reading the stamped encoding, dispatching to the codec transcode
// primitive, and returning JSON bytes for the SSE Data field. This is the
// per-event-per-client cost when WithPayloadTransform(CBORToJSONTransform) is
// wired (ADR-0052). It composes event.PayloadReadOnly + codec.TranscodeToJSON,
// so its cost is dominated by the underlying transcode (see
// codec.BenchmarkTranscodeToJSON_CBOR_To_JSON).
func BenchmarkCBORToJSONTransform_SSEWire(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()
	payload := map[string]any{
		"id":     "01HQ3TS7HNW3K4PR9XJ8Z2V5MS",
		"name":   "Alice",
		"email":  "alice@example.com",
		"count":  42,
		"active": true,
	}

	evt, err := event.New("user.created", streamID, "User", 1, payload,
		event.WithCodec(codec.CBORCodec{}))
	if err != nil {
		b.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		b.Fatalf("encoding = %q, want cbor", evt.Encoding())
	}

	b.ResetTimer()

	for b.Loop() {
		out := CBORToJSONTransform(evt)
		if len(out) == 0 {
			b.Fatal("empty transform output")
		}
	}
}

// BenchmarkCBORToJSONTransform_JSON_Passthrough measures the zero-cost path
// for JSON-encoded events: no transcode work, just a payload read. JSON-only
// deployments pay only this cost when the transform is wired.
func BenchmarkCBORToJSONTransform_JSON_Passthrough(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()
	payload := map[string]any{"id": "01HQ3TS7HNW3K4PR9XJ8Z2V5MS", "name": "Alice"}

	evt, err := event.New("user.created", streamID, "User", 1, payload,
		event.WithCodec(codec.JSONCodec{}))
	if err != nil {
		b.Fatalf("event.New: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		out := CBORToJSONTransform(evt)
		if len(out) == 0 {
			b.Fatal("empty transform output")
		}
	}
}

// BenchmarkCBORToJSONTransform_FanOut_100Clients quantifies the per-event fan-out
// cost when 100 SSE clients receive the same CBOR event. The SSE broker calls
// payloadForWire once per client (not once per event — see sse.go line 264), so
// the transform runs N times for N connected clients. This benchmark documents
// that redundancy and provides a baseline for any future memoization
// optimization (e.g., sync.OnceValue keyed by event ID).
func BenchmarkCBORToJSONTransform_FanOut_100Clients(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()
	payload := map[string]any{
		"id":     "01HQ3TS7HNW3K4PR9XJ8Z2V5MS",
		"name":   "Alice",
		"email":  "alice@example.com",
		"count":  42,
		"active": true,
	}

	evt, err := event.New("user.created", streamID, "User", 1, payload,
		event.WithCodec(codec.CBORCodec{}))
	if err != nil {
		b.Fatalf("event.New: %v", err)
	}

	const clients = 100

	b.ResetTimer()

	for b.Loop() {
		for i := 0; i < clients; i++ {
			out := CBORToJSONTransform(evt)
			if len(out) == 0 {
				b.Fatal("empty transform output")
			}
		}
	}
}
