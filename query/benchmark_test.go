package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		_, err := query.New("bench.query")
		if err != nil {
			b.Fatalf("New: %v", err)
		}
	}
}

func BenchmarkMustNew(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		_ = querytest.New(b, "bench.query")
	}
}

func BenchmarkDispatchTyped(b *testing.B) {
	b.ReportAllocs()

	d := query.NewDispatcher()
	b.Cleanup(func() { _ = d.Close() })

	err := d.Register("bench.query", func(_ context.Context, _ query.Query) (any, error) {
		return 42, nil
	})
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	ctx := context.Background()
	q := querytest.New(b, "bench.query")

	b.ResetTimer()

	for b.Loop() {
		_, err := query.DispatchTyped[int](ctx, d, q)
		if err != nil {
			b.Fatalf("DispatchTyped: %v", err)
		}
	}
}

func BenchmarkNewPagination(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		_ = query.NewPagination(1, 20)
	}
}

func BenchmarkNewPaginatedResult(b *testing.B) {
	b.ReportAllocs()

	data := make([]string, 20)
	for i := range data {
		data[i] = "item"
	}

	p := query.NewPagination(1, 20)

	b.ResetTimer()

	for b.Loop() {
		_ = query.NewPaginatedResult(data, 100, p)
	}
}

func BenchmarkPagination_Validate(b *testing.B) {
	b.ReportAllocs()

	p := query.NewPagination(1, 20)

	b.ResetTimer()

	for b.Loop() {
		_ = p.Validate()
	}
}
