package codec_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// BenchmarkTranscodeToJSON_CBOR_To_JSON measures the transcode hot path: a
// realistic CBOR-encoded event payload decoded into a generic value and
// re-encoded as JSON. This is the per-event cost SSE fan-out pays when
// CBORToJSONTransform is wired (ADR-0048).
func BenchmarkTranscodeToJSON_CBOR_To_JSON(b *testing.B) {
	b.ReportAllocs()

	cborData, err := (codec.CBORCodec{}).Encode(sampleOrder())
	if err != nil {
		b.Fatalf("encode CBOR: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := codec.TranscodeToJSON(cborData, codec.EncodingCBOR)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTranscodeToJSON_JSON_Passthrough measures the zero-cost fast path:
// non-CBOR encodings return the input unchanged with no allocation. The
// contrast with the CBOR path quantifies the overhead a JSON-only deployment
// avoids.
func BenchmarkTranscodeToJSON_JSON_Passthrough(b *testing.B) {
	b.ReportAllocs()

	jsonData, err := (codec.JSONCodec{}).Encode(sampleOrder())
	if err != nil {
		b.Fatalf("encode JSON: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		out, err := codec.TranscodeToJSON(jsonData, codec.EncodingJSON)
		if err != nil {
			b.Fatal(err)
		}

		_ = out
	}
}
