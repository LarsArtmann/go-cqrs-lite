package http

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// fuzzCodec stamps EncodingCBOR and returns a fixed byte slice on Encode,
// allowing the fuzz test to inject arbitrary CBOR data into an event payload
// without depending on event internals.
type fuzzCodec struct{ data []byte }

func (f fuzzCodec) Encoding() codec.Encoding   { return codec.EncodingCBOR }
func (f fuzzCodec) Encode(any) ([]byte, error) { return f.data, nil }
func (f fuzzCodec) Decode([]byte, any) error   { return nil }

// FuzzCBORToJSONTransform exercises the end-to-end transform path with
// arbitrary CBOR bytes: it must never panic and must always return a non-nil
// []byte (either transcoded JSON or the raw fallback). This complements the
// codec-level FuzzTranscodeToJSON by testing through the event.Event →
// PayloadReadOnly → TranscodeToJSON → slog fallback chain.
func FuzzCBORToJSONTransform(f *testing.F) {
	// Valid CBOR: map {"a": 1}
	f.Add([]byte{0xa1, 0x61, 0x61, 0x01})
	// Invalid CBOR: map header with garbage trailing
	f.Add([]byte{0xa1, 0xff, 0xff})
	// Empty payload
	f.Add([]byte{})
	// Single int 0
	f.Add([]byte{0x00})
	// Valid CBOR: array [1, 2, 3]
	f.Add([]byte{0x83, 0x01, 0x02, 0x03})
	// Deeply nested map
	f.Add(
		[]byte{
			0xa1,
			0x63,
			0x6b,
			0x65,
			0x79,
			0xa1,
			0x63,
			0x6b,
			0x65,
			0x79,
			0xa1,
			0x63,
			0x6b,
			0x65,
			0x79,
			0x01,
		},
	)

	f.Fuzz(func(t *testing.T, payload []byte) {
		t.Parallel()

		evt, err := event.New(
			"fuzz.event",
			id.NewStreamID(),
			"Fuzz",
			1,
			map[string]any{"x": 1},
			event.WithCodec(fuzzCodec{data: payload}),
		)
		if err != nil {
			// event.New can only fail if validateEventParams rejects the
			// fuzz-derived payload (e.g., nil data from empty fuzz input).
			// The fuzzCodec itself always returns nil error. A bare return
			// is the standard Go fuzz pattern for "input not applicable."
			return
		}

		// The transform must never panic.
		out := CBORToJSONTransform(evt)

		// Output must always be non-nil — SSE clients always receive data.
		if out == nil {
			t.Error("CBORToJSONTransform returned nil; expected non-nil []byte")
		}

		// For non-empty payloads, the output should also be non-empty.
		if len(payload) > 0 && len(out) == 0 {
			t.Error("CBORToJSONTransform returned empty output for non-empty payload")
		}
	})
}
