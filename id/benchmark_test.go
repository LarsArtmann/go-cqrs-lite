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
		_, _ = Parse[StreamID](validID)
	}
}

func BenchmarkParse_Invalid(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Parse[StreamID]("")
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
		_, _ = streamID.MarshalJSON()
	}
}

func BenchmarkMarshalText(b *testing.B) {
	b.ReportAllocs()
	streamID := New[StreamID]()

	for b.Loop() {
		_, _ = streamID.MarshalText()
	}
}
