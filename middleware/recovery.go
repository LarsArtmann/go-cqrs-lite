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
	return fmt.Errorf("%w: panic recovered in %s %s: %v\n%s", //nolint:err113
		ErrPanicRecovered,
		msgKind,
		typeName,
		r,
		debug.Stack(),
	)
}

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = panicError("command", string(cmd.Type()), r)
				}
			}()

			return next(ctx, cmd)
		}
	}
}

// EventRecovery returns an event middleware that recovers from panics.
func EventRecovery() event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = panicError("event", string(evt.Type()), r)
				}
			}()

			return next(ctx, evt)
		}
	}
}

// QueryRecovery returns a query middleware that recovers from panics.
func QueryRecovery() query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (result any, err error) { //nolint:nonamedreturns
			defer func() {
				if r := recover(); r != nil {
					err = panicError("query", string(q.Type()), r)
				}
			}()

			return next(ctx, q)
		}
	}
}
