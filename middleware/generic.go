package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// Handler is a generic handler for messages of type M.
// Command and event handlers both satisfy this signature (returning only error).
type Handler[M any] func(context.Context, M) error

// Middleware wraps a Handler with cross-cutting concerns.
type Middleware[M any] func(Handler[M]) Handler[M]

// MessageAdapter provides message-specific extraction for generic middleware.
type MessageAdapter[M any] struct {
	Kind        string                 // "command", "event", "query"
	ExtractType func(M) string         // extracts the message type name
	ExtractID   func(M) id.AggregateID // extracts the aggregate ID (may be nil for queries)
}

const (
	kindCommand = "command"
	kindEvent   = "event"
	kindQuery   = "query"
)

// Pre-built adapters for each CQRS message type.
var (
	//nolint:gochecknoglobals // immutable adapter, used throughout package
	CommandAdapter = MessageAdapter[command.Command]{
		Kind:        kindCommand,
		ExtractType: func(cmd command.Command) string { return string(cmd.Type()) },
		ExtractID:   func(cmd command.Command) id.AggregateID { return cmd.AggregateID() },
	}

	//nolint:gochecknoglobals // immutable adapter, used throughout package
	EventAdapter = MessageAdapter[event.Event]{
		Kind:        kindEvent,
		ExtractType: func(evt event.Event) string { return string(evt.Type()) },
		ExtractID:   func(evt event.Event) id.AggregateID { return evt.AggregateID() },
	}

	//nolint:gochecknoglobals // immutable adapter, used throughout package
	QueryAdapter = MessageAdapter[query.Query]{ //nolint:exhaustruct // queries have no aggregateID
		Kind:        kindQuery,
		ExtractType: func(q query.Query) string { return string(q.Type()) },
	}
)

// failingMiddleware returns a middleware that always fails with the given error.
func failingMiddleware[M any](err error) Middleware[M] {
	return func(Handler[M]) Handler[M] {
		return func(_ context.Context, _ M) error {
			return err
		}
	}
}

// AsCommand converts a generic Middleware to a command.Middleware.
func AsCommand(mw Middleware[command.Command]) command.Middleware {
	return func(next command.Handler) command.Handler {
		h := mw(Handler[command.Command](next))

		return command.Handler(h)
	}
}

// AsEvent converts a generic Middleware to an event.Middleware.
func AsEvent(mw Middleware[event.Event]) event.Middleware {
	return func(next event.Handler) event.Handler {
		h := mw(Handler[event.Event](next))

		return event.Handler(h)
	}
}

// AsQuery wraps a generic error-only Middleware for use with query handlers.
// It captures the result from the query handler and propagates it through.
func AsQuery(middleware Middleware[query.Query]) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			var result any

			errOnly := func(_ context.Context, _ query.Query) error {
				var err error

				result, err = next(ctx, q)

				return err
			}

			wrapped := middleware(errOnly)

			err := wrapped(ctx, q)

			return result, err
		}
	}
}
