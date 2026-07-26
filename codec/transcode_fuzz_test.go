package codec_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// FuzzTranscodeToJSON asserts the transcode path never panics on arbitrary
// input. For EncodingCBOR, random bytes must resolve to either a successful
// JSON result or a typed error — never a crash. Non-CBOR encodings must return
// the input bytes unchanged (the documented passthrough contract).
func FuzzTranscodeToJSON(f *testing.F) {
	// Seed with: valid CBOR map, invalid CBOR, empty, JSON-looking bytes.
	f.Add([]byte{0xa1, 0x63, 'k', 0x18, 0x2a}, string(codec.EncodingCBOR))
	f.Add([]byte{0xa1, 0xff, 0xff}, string(codec.EncodingCBOR))
	f.Add([]byte{}, string(codec.EncodingCBOR))
	f.Add([]byte(`{"x":1}`), string(codec.EncodingJSON))
	f.Add([]byte(`not-cbor`), string(codec.EncodingRaw))
	f.Add([]byte{0x00}, string(codec.EncodingCBOR))

	f.Fuzz(func(t *testing.T, payload []byte, encStr string) {
		t.Parallel()

		enc := codec.Encoding(encStr)
		out, err := codec.TranscodeToJSON(payload, enc)
		if err != nil {
			// On error, the contract is that callers fall back to the raw
			// payload (CBORToJSONTransform does this). No output is trusted.
			return
		}

		// Non-CBOR encodings pass through byte-identical.
		if enc != codec.EncodingCBOR {
			if string(out) != string(payload) {
				t.Fatalf("passthrough mismatch: enc=%s got %q want %q", enc, out, payload)
			}

			return
		}

		// CBOR success path: the output must be valid JSON. Generic decode +
		// json.Marshal should always yield parseable JSON (numbers, strings,
		// arrays, objects, bools, null). Unmarshal is the v2 validity probe.
		var probe any
		if err := json.Unmarshal(out, &probe); err != nil {
			t.Fatalf("transcode produced invalid JSON: %q (err: %v)", out, err)
		}
	})
}
