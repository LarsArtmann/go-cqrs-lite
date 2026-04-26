package middleware

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandValidation returns a middleware that validates commands before dispatch.
func CommandValidation(validate Validator) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			err := validate(cmd)
			if err != nil {
				return fmt.Errorf("validation failed for command %s: %w", cmd.Type(), err)
			}

			return next(ctx, cmd)
		}
	}
}

// EventValidation returns a middleware that validates events before handling.
func EventValidation(validate Validator) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			err := validate(evt)
			if err != nil {
				return fmt.Errorf("validation failed for event %s: %w", evt.Type(), err)
			}

			return next(ctx, evt)
		}
	}
}

// QueryValidation returns a middleware that validates queries before dispatch.
func QueryValidation(validate Validator) query.Middleware {
	return func(next func(context.Context, query.Query) (any, error)) func(context.Context, query.Query) (any, error) {
		return func(ctx context.Context, q query.Query) (any, error) {
			err := validate(q)
			if err != nil {
				return nil, fmt.Errorf("validation failed for query %s: %w", q.Type(), err)
			}

			return next(ctx, q)
		}
	}
}
