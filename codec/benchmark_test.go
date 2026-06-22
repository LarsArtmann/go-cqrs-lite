package codec_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

func BenchmarkJSONCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.JSONCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.JSONCodec{}
	data, _ := c.Encode(map[string]string{"name": "Alice", "email": "alice@example.com"})

	b.ResetTimer()

	for b.Loop() {
		var result map[string]string
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.CBORCodec{}
	data, _ := c.Encode(map[string]string{"name": "Alice", "email": "alice@example.com"})

	b.ResetTimer()

	for b.Loop() {
		var result map[string]string
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecComparison_Encode(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := jsonCodec.Encode(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, err := cborCodec.Encode(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCodecComparison_Decode(b *testing.B) {
	b.ReportAllocs()

	jsonCodec := codec.JSONCodec{}
	cborCodec := codec.CBORCodec{}
	payload := map[string]string{"name": "Alice", "email": "alice@example.com"}

	jsonData, _ := jsonCodec.Encode(payload)
	cborData, _ := cborCodec.Encode(payload)

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result map[string]string
			if err := jsonCodec.Decode(jsonData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CBOR", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			var result map[string]string
			if err := cborCodec.Decode(cborData, &result); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRawCodec_Encode(b *testing.B) {
	b.ReportAllocs()

	c := codec.RawCodec{}
	data := []byte("raw payload bytes")

	b.ResetTimer()

	for b.Loop() {
		_, err := c.Encode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRawCodec_Decode(b *testing.B) {
	b.ReportAllocs()

	c := codec.RawCodec{}
	data := []byte("raw payload bytes")

	b.ResetTimer()

	for b.Loop() {
		var result []byte
		if err := c.Decode(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCBORCompact_vs_Canon_Size(b *testing.B) {
	type eventPayload struct {
		Name    string
		Email   string
		Version int
		Active  bool
	}
	payload := eventPayload{Name: "Alice", Email: "alice@example.com", Version: 42, Active: true}

	canonical := codec.CBORCodec{}
	compact := codec.CBORCompactCodec{}

	canonicalData, _ := canonical.Encode(payload)
	compactData, _ := compact.Encode(payload)

	b.Logf(
		"CBOR (canonical): %d bytes, CBOR (compact): %d bytes, savings: %.1f%%",
		len(canonicalData), len(compactData),
		float64(len(canonicalData)-len(compactData))/float64(len(canonicalData))*100,
	)

	codecs := []struct {
		name string
		c    codec.Codec
	}{
		{"Canonical", canonical},
		{"Compact", compact},
	}

	for _, tc := range codecs {
		b.Run(tc.name+"/Encode", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := tc.c.Encode(payload)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
