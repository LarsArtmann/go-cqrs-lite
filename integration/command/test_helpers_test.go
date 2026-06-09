package command_test

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
)

func noopCommandHandler() command.Handler {
	var h command.Handler = func(_ context.Context, _ command.Command) error { return nil }

	return h
}

func callbackCommandHandler(flag *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*flag = true

		return nil
	}
}

func appendCommandHandler(order *[]string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*order = append(*order, "handler")

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
