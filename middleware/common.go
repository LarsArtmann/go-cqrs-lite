package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func commandErrMiddleware(err error) command.Middleware {
	return func(_ command.Handler) command.Handler {
		return func(_ context.Context, _ command.Command) error {
			return err
		}
	}
}

func eventErrMiddleware(err error) event.Middleware {
	return func(_ event.Handler) event.Handler {
		return func(_ context.Context, _ event.Event) error {
			return err
		}
	}
}

func queryErrMiddleware(err error) query.Middleware {
	return func(_ query.Handler) query.Handler {
		return func(_ context.Context, _ query.Query) (any, error) {
			return nil, err
		}
	}
}
