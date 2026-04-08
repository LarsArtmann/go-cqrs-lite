package id

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for range b.N {
		New[AggregateID]()
	}
}

func BenchmarkNewWithPrefix(b *testing.B) {
	for range b.N {
		NewWithPrefix[AggregateID]("agg")
	}
}

func BenchmarkParse(b *testing.B) {
	validID := "550e8400-e29b-41d4-a716-446655440000"

	b.ResetTimer()

	for range b.N {
		_, _ = Parse[AggregateID](validID)
	}
}

func BenchmarkParse_Invalid(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_, _ = Parse[AggregateID]("")
	}
}

func BenchmarkString(b *testing.B) {
	aggregateID := New[AggregateID]()

	b.ResetTimer()

	for range b.N {
		_ = aggregateID.String()
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	aggregateID := New[AggregateID]()

	b.ResetTimer()

	for range b.N {
		_, _ = aggregateID.MarshalJSON()
	}
}

func BenchmarkMarshalText(b *testing.B) {
	aggregateID := New[AggregateID]()

	b.ResetTimer()

	for range b.N {
		_, _ = aggregateID.MarshalText()
	}
}
