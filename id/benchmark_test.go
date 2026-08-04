package id

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		New[StreamID]()
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	validID := "01HK1549P84T9XF8R94E960633"

	for b.Loop() {
		parsed, err := Parse[StreamID](validID)
		if err != nil {
			b.Fatalf("Parse: %v", err)
		}
		if parsed.String() != validID {
			b.Fatalf("Parse: got %q, want %q", parsed.String(), validID)
		}
	}
}

func BenchmarkParse_Invalid(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, err := Parse[StreamID]("")
		if err == nil {
			b.Fatal("Parse(\"\"): expected error for invalid input")
		}
	}
}

func BenchmarkString(b *testing.B) {
	b.ReportAllocs()
	streamID := New[StreamID]()

	for b.Loop() {
		_ = streamID.String()
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	b.ReportAllocs()
	streamID := New[StreamID]()

	for b.Loop() {
		data, err := streamID.MarshalJSON()
		if err != nil {
			b.Fatalf("MarshalJSON: %v", err)
		}
		if len(data) == 0 {
			b.Fatal("MarshalJSON: returned empty bytes")
		}
	}
}

func BenchmarkMarshalText(b *testing.B) {
	b.ReportAllocs()
	streamID := New[StreamID]()

	for b.Loop() {
		data, err := streamID.MarshalText()
		if err != nil {
			b.Fatalf("MarshalText: %v", err)
		}
		if len(data) == 0 {
			b.Fatal("MarshalText: returned empty bytes")
		}
	}
}
