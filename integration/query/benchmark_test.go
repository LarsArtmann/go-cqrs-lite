package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	dispatcher := query.NewDispatcher()

	err := dispatcher.Register("bench.query", noopQueryHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	q := query.MustNew("bench.query")
	ctx := context.Background()

	for b.Loop() {
		_, err := dispatcher.Dispatch(ctx, q)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}

func BenchmarkDispatcher_Dispatch_WithMiddleware(b *testing.B) {
	dispatcher := query.NewDispatcher()

	middleware := queryMiddleware(nil, "middleware")

	dispatcher.Use(middleware, middleware)

	err := dispatcher.Register("bench.query", noopQueryHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	q := query.MustNew("bench.query")
	ctx := context.Background()

	for b.Loop() {
		_, err := dispatcher.Dispatch(ctx, q)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}
