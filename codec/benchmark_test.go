package codec_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
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
