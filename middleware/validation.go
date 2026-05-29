package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandValidation returns a middleware that validates commands before dispatch.
func CommandValidation(validate CommandValidator, opts ...Option) command.Middleware {
	cfg := applyOptions(opts)

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			err := validate(cmd)
			if err != nil {
				if cfg.logger != nil {
					cfg.logger.Warn(
						"validation failed",
						"kind", "command",
						"type", cmd.Type(),
						"error", err,
					)
				}

				return event.Wrapf(
					ErrValidationFailed, event.Rejection,
					"middleware.command_validation",
					"validation failed for command %s",
					cmd.Type(),
				)
			}

			return next(ctx, cmd)
		}
	}
}

// EventValidation returns a middleware that validates events before handling.
func EventValidation(validate EventValidator, opts ...Option) event.Middleware {
	cfg := applyOptions(opts)

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			err := validate(evt)
			if err != nil {
				if cfg.logger != nil {
					cfg.logger.Warn(
						"validation failed",
						"kind", "event",
						"type", evt.Type(),
						"error", err,
					)
				}

				return event.Wrapf(
					ErrValidationFailed, event.Rejection,
					"middleware.event_validation",
					"validation failed for event %s",
					evt.Type(),
				)
			}

			return next(ctx, evt)
		}
	}
}

// QueryValidation returns a middleware that validates queries before dispatch.
func QueryValidation(validate QueryValidator, opts ...Option) query.Middleware {
	cfg := applyOptions(opts)

	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			err := validate(q)
			if err != nil {
				if cfg.logger != nil {
					cfg.logger.Warn(
						"validation failed",
						"kind", "query",
						"type", q.Type(),
						"error", err,
					)
				}

				return nil, event.Wrapf(
					ErrValidationFailed, event.Rejection,
					"middleware.query_validation",
					"validation failed for query %s",
					q.Type(),
				)
			}

			return next(ctx, q)
		}
	}
}
