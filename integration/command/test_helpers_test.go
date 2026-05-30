package command_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command"
)

func noopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

func callbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

func appendCommandHandler(callOrder *[]string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*callOrder = append(*callOrder, "handler")

		return nil
	}
}

func commandMiddleware(callOrder *[]string, name string) func(h command.Handler) command.Handler {
	return func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, cmd)
		}
	}
}
