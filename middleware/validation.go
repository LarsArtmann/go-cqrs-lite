package middleware

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// NewValidation returns a generic middleware that validates messages before dispatch.
func NewValidation[M any](
	adapter MessageAdapter[M],
	validate func(M) error,
	opts ...Option,
) Middleware[M] {
	cfg := applyOptions(opts)

	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			err := validate(msg)
			if err != nil {
				if cfg.logger != nil {
					cfg.logger.Warn(
						"validation failed",
						"kind", adapter.Kind,
						"type", adapter.ExtractType(msg),
						"error", err,
					)
				}

				return errorfamily.Wrapf(
					err, errorfamily.Rejection,
					"middleware."+adapter.Kind+"_validation",
					"validation failed for %s %s",
					adapter.Kind,
					adapter.ExtractType(msg),
				)
			}

			return next(ctx, msg)
		}
	}
}

// CommandValidation returns a middleware that validates commands before dispatch.
func CommandValidation(validate CommandValidator, opts ...Option) command.Middleware {
	return AsCommand(NewValidation(CommandAdapter, func(cmd command.Command) error {
		return validate(cmd)
	}, opts...))
}

// EventValidation returns a middleware that validates events before handling.
func EventValidation(validate EventValidator, opts ...Option) event.Middleware {
	return AsEvent(NewValidation(EventAdapter, func(evt event.Event) error {
		return validate(evt)
	}, opts...))
}

// QueryValidation returns a middleware that validates queries before dispatch.
func QueryValidation(validate QueryValidator, opts ...Option) query.Middleware {
	return AsQuery(NewValidation(QueryAdapter, func(q query.Query) error {
		return validate(q)
	}, opts...))
}
