package query_test

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/query"
)

func failingQueryHandler(msg string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New(msg)
	}
}
