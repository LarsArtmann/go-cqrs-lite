package query_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func noopQueryHandler() query.Handler {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, nil
	}
}

func queryMiddleware(callOrder *[]string, name string) query.Middleware {
	return func(h query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			if callOrder != nil {
				*callOrder = append(*callOrder, name)
			}

			return h(ctx, q)
		}
	}
}
