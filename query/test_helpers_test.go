package query_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func failingQueryHandler(msg string) query.Handler {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New(msg)
	}
}
