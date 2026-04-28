package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery() command.Middleware {
	return func(next command.Handler) command.Handler {
		//nolint:nonamedreturns // required for defer/recover to modify return values
		return func(ctx context.Context, cmd command.Command) (err error) {
			defer func() {
				if r := recover(); r != nil {
					//nolint:err113
					err = fmt.Errorf("panic recovered in command %s: %v\n%s", cmd.Type(), r, debug.Stack())
				}
			}()

			return next(ctx, cmd)
		}
	}
}

// EventRecovery returns an event middleware that recovers from panics.
func EventRecovery() event.Middleware {
	return func(next event.Handler) event.Handler {
		//nolint:nonamedreturns // required for defer/recover to modify return values
		return func(ctx context.Context, evt event.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					//nolint:err113
					err = fmt.Errorf("panic recovered in event %s: %v\n%s", evt.Type(), r, debug.Stack())
				}
			}()

			return next(ctx, evt)
		}
	}
}

// QueryRecovery returns a query middleware that recovers from panics.
func QueryRecovery() query.Middleware {
	return func(next query.Handler) query.Handler {
		//nolint:nonamedreturns // required for defer/recover to modify return values
		return func(ctx context.Context, q query.Query) (result any, err error) {
			defer func() {
				if r := recover(); r != nil {
					//nolint:err113
					err = fmt.Errorf("panic recovered in query %s: %v\n%s", q.Type(), r, debug.Stack())
				}
			}()

			return next(ctx, q)
		}
	}
}
