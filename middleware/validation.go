package middleware

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
)

// CommandValidation returns a middleware that validates commands before dispatch.
func CommandValidation(validate Validator) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			if err := validate(cmd); err != nil {
				return fmt.Errorf("validation failed for command %s: %w", cmd.Type(), err)
			}

			return next(ctx, cmd)
		}
	}
}

// QueryValidation returns a middleware that validates queries before dispatch.
func QueryValidation(validate Validator) query.Middleware {
	return func(next func(query.Query) (any, error)) func(query.Query) (any, error) {
		return func(q query.Query) (any, error) {
			if err := validate(q); err != nil {
				return nil, fmt.Errorf("validation failed for query %s: %w", q.Type(), err)
			}

			return next(q)
		}
	}
}
