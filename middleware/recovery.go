package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// panicError formats a recovered panic into an error message.
func panicError(msgKind, typeName string, r any) error {
	return fmt.Errorf(
		"%w: panic recovered in %s %s: %v\n%s",
		ErrPanicRecovered,
		msgKind,
		typeName,
		r,
		debug.Stack(),
	)
}

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery(opts ...Option) command.Middleware {
	cfg := applyOptions(opts)

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = panicError("command", string(cmd.Type()), r)

					if cfg.logger != nil {
						cfg.logger.Error(
							"panic recovered",
							"kind", "command",
							"type", cmd.Type(),
							"panic", r,
						)
					}
				}
			}()

			return next(ctx, cmd)
		}
	}
}

// EventRecovery returns an event middleware that recovers from panics.
func EventRecovery(opts ...Option) event.Middleware {
	cfg := applyOptions(opts)

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = panicError("event", string(evt.Type()), r)

					if cfg.logger != nil {
						cfg.logger.Error(
							"panic recovered",
							"kind", "event",
							"type", evt.Type(),
							"panic", r,
						)
					}
				}
			}()

			return next(ctx, evt)
		}
	}
}

// QueryRecovery returns a query middleware that recovers from panics.
func QueryRecovery(opts ...Option) query.Middleware {
	cfg := applyOptions(opts)

	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (result any, err error) { //nolint:nonamedreturns
			defer func() {
				if r := recover(); r != nil {
					err = panicError("query", string(q.Type()), r)

					if cfg.logger != nil {
						cfg.logger.Error(
							"panic recovered",
							"kind", "query",
							"type", q.Type(),
							"panic", r,
						)
					}
				}
			}()

			return next(ctx, q)
		}
	}
}
