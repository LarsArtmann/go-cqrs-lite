package id

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		New[AggregateID]()
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	validID := "01HK1549P84T9XF8R94E960633"

	for b.Loop() {
		_, _ = Parse[AggregateID](validID)
	}
}

func BenchmarkParse_Invalid(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Parse[AggregateID]("")
	}
}

func BenchmarkString(b *testing.B) {
	b.ReportAllocs()
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_ = aggregateID.String()
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	b.ReportAllocs()
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_, _ = aggregateID.MarshalJSON()
	}
}

func BenchmarkMarshalText(b *testing.B) {
	b.ReportAllocs()
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_, _ = aggregateID.MarshalText()
	}
}
