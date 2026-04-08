package id

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		New[AggregateID]()
	}
}

func BenchmarkNewWithPrefix(b *testing.B) {
	for b.Loop() {
		NewWithPrefix[AggregateID]("agg")
	}
}

func BenchmarkParse(b *testing.B) {
	validID := "550e8400-e29b-41d4-a716-446655440000"

	for b.Loop() {
		_, _ = Parse[AggregateID](validID)
	}
}

func BenchmarkParse_Invalid(b *testing.B) {
	for b.Loop() {
		_, _ = Parse[AggregateID]("")
	}
}

func BenchmarkString(b *testing.B) {
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_ = aggregateID.String()
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_, _ = aggregateID.MarshalJSON()
	}
}

func BenchmarkMarshalText(b *testing.B) {
	aggregateID := New[AggregateID]()

	for b.Loop() {
		_, _ = aggregateID.MarshalText()
	}
}
