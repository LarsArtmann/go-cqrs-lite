package metaengine

import (
	"encoding/json/v2"
	"math/rand/v2"
	"testing"
)

// Codec micro-benchmarks pin the pure encode/decode cost of the binary
// vector payload against the legacy JSON array, independent of any engine's
// scan path (docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md §2).
// The reference point for future format work (int8 quantization, Phase 1):
// D=128 is the spike's working dimension, D=1536 the OpenAI-embedding-class
// dimension.

func benchVector(dim int) []float32 {
	rng := rand.New(rand.NewPCG(42, 7))
	values := make([]float32, dim)

	for i := range values {
		values[i] = rng.Float32()*2 - 1
	}

	return values
}

func benchmarkDecodeVector(b *testing.B, dim int) {
	b.Helper()

	values := benchVector(dim)
	binaryPayload := EncodeVectorBinary(values)

	jsonPayload, err := json.Marshal(values)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}

	b.Run("binary", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if _, err := DecodeVectorBinary(binaryPayload); err != nil {
				b.Fatalf("DecodeVectorBinary: %v", err)
			}
		}
	})

	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if _, err := DecodeVectorJSON(jsonPayload); err != nil {
				b.Fatalf("DecodeVectorJSON: %v", err)
			}
		}
	})
}

func BenchmarkDecodeVector_D128(b *testing.B)  { benchmarkDecodeVector(b, 128) }
func BenchmarkDecodeVector_D1536(b *testing.B) { benchmarkDecodeVector(b, 1536) }

func BenchmarkEncodeVectorBinary_D128(b *testing.B)  { benchmarkEncodeVector(b, 128) }
func BenchmarkEncodeVectorBinary_D1536(b *testing.B) { benchmarkEncodeVector(b, 1536) }

func benchmarkEncodeVector(b *testing.B, dim int) {
	b.Helper()

	values := benchVector(dim)
	b.ReportAllocs()

	for b.Loop() {
		_ = EncodeVectorBinary(values)
	}
}
