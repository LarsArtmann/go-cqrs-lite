package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	dispatcher := query.NewDispatcher()

	err := dispatcher.Register("bench.query", testhelpers.NoopQueryHandler())
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

	middleware := func(next func(context.Context, query.Query) (any, error)) func(context.Context, query.Query) (any, error) {
		return func(ctx context.Context, q query.Query) (any, error) {
			return next(ctx, q)
		}
	}

	dispatcher.Use(middleware, middleware)

	err := dispatcher.Register("bench.query", testhelpers.NoopQueryHandler())
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
