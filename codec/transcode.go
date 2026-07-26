package codec

import (
	"encoding/json/v2"
	"fmt"
)

// TranscodeToJSON converts a payload from its stamped encoding into JSON bytes.
// It is the generic, schema-free bridge for consumers that must serve JSON to
// clients (browsers, REST APIs) while storing events in a compact encoding
// (CBOR). This deletes the per-consumer CBOR→JSON transcode logic that every
// compact-codec deployment otherwise duplicates.
//
// Behaviour by encoding:
//   - [EncodingJSON]: payload returned unchanged (already JSON).
//   - [EncodingRaw]: payload returned unchanged (caller asserts it is valid JSON).
//   - [EncodingCBOR]: payload is decoded into a generic Go value
//     (map/slice/scalar) and re-encoded as JSON, preserving the CBOR data model.
//
// TranscodeToJSON is schema-free: it does NOT know the original Go struct type.
// CBOR maps become JSON objects; CBOR arrays — including structs encoded with
// the cbor:",toarray" tag — become JSON arrays. For schema-aware decoding (field
// names, typed values), use event.DecodePayloadAuto[T] with the concrete type.
//
// An error is returned only when CBOR decoding or JSON encoding fails. Callers
// that want graceful degradation (fall back to the raw payload on failure)
// should ignore the error and use the original bytes — see
// transport/http.CBORToJSONTransform for a ready-made [WithPayloadTransform]
// adapter that does exactly this.
func TranscodeToJSON(payload []byte, enc Encoding) ([]byte, error) {
	if enc != EncodingCBOR {
		return payload, nil
	}

	var v any
	if err := canonicalDecMode().Unmarshal(payload, &v); err != nil {
		return nil, fmt.Errorf("codec: decode CBOR for transcode: %w", err)
	}

	out, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("codec: encode JSON for transcode: %w", err)
	}

	return out, nil
}
