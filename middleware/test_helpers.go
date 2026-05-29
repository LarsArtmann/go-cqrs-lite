package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command"
)

// NoopCommandHandler returns a handler that always returns nil.
func NoopCommandHandler() func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}
