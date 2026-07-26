package http

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// CBORToJSONTransform is a ready-made payload transform for [WithPayloadTransform]
// that converts CBOR-encoded event payloads to JSON for browser consumption.
// Non-CBOR payloads (JSON, Raw) pass through unchanged with zero overhead.
//
// On any decode/encode failure, the original raw payload is returned unchanged
// (graceful degradation) so SSE clients always receive data — never a gap. This
// deletes the per-consumer CBOR→JSON transcode logic that every compact-codec
// deployment otherwise duplicates (~50 LOC of memoized decoders, decode/re-encode,
// and fallback handling).
//
// Recommended one-liner for consumers using CBOR as their default event codec:
//
//	broker, _ := NewSSEBroker(bus, WithPayloadTransform(CBORToJSONTransform))
//
// Transcoding is schema-free (generic): CBOR maps become JSON objects and CBOR
// arrays — including structs encoded with the cbor:",toarray" tag — stay arrays.
// For schema-aware JSON output (reconstructing field names from toarray structs),
// pass a custom transform that uses event.DecodePayloadAuto[T] with the concrete
// payload type.
func CBORToJSONTransform(evt event.Event) []byte {
	raw := event.PayloadReadOnly(evt)

	out, err := codec.TranscodeToJSON(raw, evt.Encoding())
	if err != nil {
		return raw
	}

	return out
}
