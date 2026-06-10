package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func mustNewQuery(queryType query.Type) *query.BasicQuery {
	q, err := query.New(queryType)
	if err != nil {
		panic(err)
	}
	return q
}

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	b.ReportAllocs()
	dispatcher := query.NewDispatcher()

	err := dispatcher.Register("bench.query", noopQueryHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	q := mustNewQuery("bench.query")
	ctx := context.Background()

	for b.Loop() {
		_, err := dispatcher.Dispatch(ctx, q)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}

func BenchmarkDispatcher_Dispatch_WithMiddleware(b *testing.B) {
	b.ReportAllocs()
	dispatcher := query.NewDispatcher()

	middleware := queryMiddleware(nil, "middleware")

	dispatcher.Use(middleware, middleware)

	err := dispatcher.Register("bench.query", noopQueryHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	q := mustNewQuery("bench.query")
	ctx := context.Background()

	for b.Loop() {
		_, err := dispatcher.Dispatch(ctx, q)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}
