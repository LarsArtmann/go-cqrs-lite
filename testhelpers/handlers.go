package testhelpers

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// AppendEventsHandler returns a bus handler that appends received events to *events.
func AppendEventsHandler(events *[]event.Event) event.Handler {
	return func(_ context.Context, evt event.Event) error {
		*events = append(*events, evt)

		return nil
	}
}

// NoopCommandHandler returns a handler that does nothing.
func NoopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

// NoopEventHandler returns a handler that does nothing.
func NoopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

// NoopEventPublisher returns a publisher that does nothing.
func NoopEventPublisher() event.Publisher {
	return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return nil
	})
}

// FailingEventPublisher returns a publisher that always returns an error.
func FailingEventPublisher(msg string) event.Publisher {
	return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return errors.New(msg) //nolint:err113 // test helper with dynamic message
	})
}

// NoopQueryHandler returns a handler that does nothing and returns nil.
func NoopQueryHandler() func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, nil
	}
}

// FailingCommandHandler returns a handler that returns an error.
func FailingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg) //nolint:err113 // test helper with dynamic message
	}
}

// FailingEventHandler returns a handler that returns an error.
func FailingEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return errors.New(msg) //nolint:err113 // test helper with dynamic message
	}
}

// FailingQueryHandler returns a query handler that returns an error.
func FailingQueryHandler(msg string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New(msg) //nolint:err113 // test helper with dynamic message
	}
}

// PanicCommandHandler returns a handler that panics with the given message.
func PanicCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		panic(msg)
	}
}

// PanicEventHandler returns a handler that panics with the given message.
func PanicEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		panic(msg)
	}
}

// PanicQueryHandler returns a query handler that panics with the given message.
func PanicQueryHandler(msg string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		panic(msg)
	}
}

// CallbackCommandHandler returns a handler that sets *called to true.
func CallbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

// CallbackEventHandler returns a handler that sets *called to true.
func CallbackEventHandler(called *bool) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		*called = true

		return nil
	}
}

// CallbackQueryHandler returns a query handler that sets *called to true.
func CallbackQueryHandler(called *bool) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		*called = true

		return nil, nil
	}
}

// CommandMiddleware creates middleware that tracks call order for command handlers.
func CommandMiddleware(callOrder *[]string, name string) func(h command.Handler) command.Handler {
	return func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, cmd)
		}
	}
}

// EventMiddleware creates middleware that tracks call order for event handlers.
func EventMiddleware(callOrder *[]string, name string) func(h event.Handler) event.Handler {
	return func(h event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, evt)
		}
	}
}

// queryHandler is a plain query handler function.
type queryHandler = func(context.Context, query.Query) (any, error)

// QueryMiddleware creates middleware that tracks call order for query handlers.
func QueryMiddleware(
	callOrder *[]string,
	name string,
) func(h func(context.Context, query.Query) (any, error)) queryHandler {
	return func(h func(context.Context, query.Query) (any, error)) queryHandler {
		return func(ctx context.Context, q query.Query) (any, error) {
			*callOrder = append(*callOrder, name)

			return h(ctx, q)
		}
	}
}
