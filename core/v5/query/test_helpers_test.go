package query_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/v5/query"
)

func failingQueryHandler(msg string) query.Handler {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New(msg)
	}
}
