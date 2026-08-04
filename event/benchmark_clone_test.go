package event

import (
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

func BenchmarkPayload(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"16B", 16},
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4096},
		{"16KB", 16 * 1024},
		{"64KB", 64 * 1024},
	}

	for _, sz := range sizes {
		payload := make([]byte, sz.n)
		for i := range payload {
			payload[i] = byte(i % 256)
		}

		b.Run("slices.Clone/"+sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = slices.Clone(payload)
			}
		})

		b.Run("make_copy/"+sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				dst := make([]byte, len(payload))
				copy(dst, payload)
				_ = dst
			}
		})

		b.Run("append/"+sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = append([]byte(nil), payload...)
			}
		})
	}
}

func BenchmarkPayload_access(b *testing.B) {
	sizes := []struct {
		name string
		n    int
	}{
		{"16B", 16},
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4096},
	}

	for _, sz := range sizes {
		payload := make([]byte, sz.n)
		for i := range payload {
			payload[i] = byte(i % 256)
		}

		evt, err := NewEvent(
			Type("UserCreated"),
			id.NewStreamID(),
			"User",
			1,
			payload,
		)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("Payload/"+sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = evt.Payload()
			}
		})

		b.Run("direct_field/"+sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = evt.payload
			}
		})
	}
}

func BenchmarkDecodePayload_clone_vs_direct(b *testing.B) {
	payload := []byte(`{"name":"Alice","email":"alice@example.com","age":30}`)

	evt, err := NewEvent(
		Type("UserCreated"),
		id.NewStreamID(),
		"User",
		1,
		payload,
	)
	if err != nil {
		b.Fatal(err)
	}

	c := codec.JSONCodec{}

	b.Run("via_Payload", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			p := evt.Payload()
			var target map[string]string
			if err := c.Decode(p, &target); err != nil {
				b.Fatalf("Decode: %v", err)
			}
			if target["name"] != "Alice" {
				b.Fatalf("Decode: target=%v, want name=Alice", target)
			}
		}
	})

	b.Run("direct_field", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			p := evt.payload
			var target map[string]string
			if err := c.Decode(p, &target); err != nil {
				b.Fatalf("Decode: %v", err)
			}
			if target["name"] != "Alice" {
				b.Fatalf("Decode: target=%v, want name=Alice", target)
			}
		}
	})

	b.Run("DecodePayload_optimized", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			result, err := DecodePayload[map[string]string](evt, c)
			if err != nil {
				b.Fatalf("DecodePayload: %v", err)
			}
			if result["name"] != "Alice" {
				b.Fatalf("DecodePayload: result=%v, want name=Alice", result)
			}
		}
	})
}

func BenchmarkMetadata_access(b *testing.B) {
	meta := Metadata{
		Tracing: metadata.Tracing{
			CorrelationID: id.NewCorrelationID(),
			CausationID:   id.NewCausationID(),
			UserID:        id.NewUserID(),
			RequestID:     id.NewRequestID(),
		},
		Source: "test-service",
		Custom: map[MetadataKey]string{
			"traceId":  "abc-123-def-456",
			"spanId":   "span-789",
			"tenantId": "tenant-42",
		},
	}

	b.Run("Clone/with_custom", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = meta.Clone()
		}
	})

	b.Run("Clone/no_custom", func(b *testing.B) {
		b.ReportAllocs()
		noCustom := meta
		noCustom.Custom = nil

		for b.Loop() {
			_ = noCustom.Clone()
		}
	})

	b.Run("value_copy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = meta
		}
	})
}

func BenchmarkPayloadReadOnly(b *testing.B) {
	payload := []byte(`{"name":"Alice","email":"alice@example.com","age":30}`)

	evt, err := NewEvent(
		Type("UserCreated"),
		id.NewStreamID(),
		"User",
		1,
		payload,
	)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Payload", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = evt.Payload()
		}
	})

	b.Run("PayloadReadOnly", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = PayloadReadOnly(evt)
		}
	})
}
