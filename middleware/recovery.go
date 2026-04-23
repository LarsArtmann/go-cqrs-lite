package middleware

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) (err error) {
			defer func() {
				if r := recover(); r != nil {
					//nolint:err113
					err = fmt.Errorf("panic recovered in command %s: %v", cmd.Type(), r)
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
					//nolint:err113
					err = fmt.Errorf("panic recovered in event %s: %v", evt.Type(), r)
				}
			}()

			return next(ctx, evt)
		}
	}
}
