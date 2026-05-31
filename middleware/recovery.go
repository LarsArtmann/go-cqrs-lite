package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/query"
)

// panicError formats a recovered panic into an error message.
func panicError(msgKind, typeName string, r any) error {
	return event.Wrapf(
		ErrPanicRecovered, event.Corruption,
		"middleware.panic_detail",
		"panic recovered in %s %s: %v",
		msgKind,
		typeName,
		r,
	)
}

func handleRecovery(cfg middlewareConfig, msgKind, typeName string, r any) error {
	err := panicError(msgKind, typeName, r)

	if cfg.logger != nil {
		cfg.logger.Error(
			"panic recovered",
			"kind", msgKind,
			"type", typeName,
			"panic", r,
		)
	}

	return err
}

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery(opts ...Option) command.Middleware {
	cfg := applyOptions(opts)

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = handleRecovery(cfg, "command", string(cmd.Type()), r)
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
					err = handleRecovery(cfg, "event", string(evt.Type()), r)
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
					err = handleRecovery(cfg, "query", string(q.Type()), r)
				}
			}()

			return next(ctx, q)
		}
	}
}
