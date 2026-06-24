package query

import (
	"context"
	"errors"
	"io"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Handler processes a query and returns a result.
//
// The return type is `any` because a single dispatcher handles heterogeneous query types
// — each query type produces a different result. This is a fundamental Go limitation:
// heterogeneous dispatch requires type erasure at the interface level (same as
// database/sql.Scan, json.Unmarshal, etc.).
//
// For type-safe dispatch, use the "typed bookend" pattern:
//   - Register side: [RegisterTyped] wraps a [TypedHandler] that returns a concrete type T.
//   - Dispatch side: [DispatchTyped] asserts the result back to T with a clear error on mismatch.
//
// This pushes the `any` ↔ T conversion to the framework boundary, giving consumers
// compile-time type safety in their handler and caller code.
//
// Deprecated: Use TypedHandler[Q, R] with RegisterTyped/DispatchTyped for compile-time
// type safety. The any-returning Handler will be replaced by a generic signature in v4.
type Handler = func(context.Context, Query) (any, error)

// Dispatcher routes queries to their handlers.
type Dispatcher struct {
	inner *dispatcher.Dispatcher[Handler, Middleware]
}

var _ io.Closer = (*Dispatcher)(nil)

// NewDispatcher creates a new query dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		inner: dispatcher.NewDispatcher[Handler, Middleware](),
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.inner.Use(middleware...)
}

// Register binds a handler to a query type.
func (d *Dispatcher) Register(queryType Type, handler Handler) error {
	err := d.ensureOpen("query.register_failed", "registering query type "+string(queryType))
	if err != nil {
		return err
	}

	err = d.inner.Register(
		string(queryType),
		handler,
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"query.register_handler_failed",
			"registering handler for query type "+string(queryType),
		)
	}

	return nil
}

// RegisterTyped binds a typed handler to a query type.
// Q is the concrete query type, R is the result type.
// The handler receives the concrete query type Q directly (no manual type assertion needed),
// and its typed result R is wrapped to match the Handler signature.
func RegisterTyped[Q Query, R any](
	d *Dispatcher,
	queryType Type,
	handler TypedHandler[Q, R],
) error {
	return d.Register(queryType, func(ctx context.Context, q Query) (any, error) {
		typed, ok := q.(Q)
		if !ok {
			return nil, ErrTypeAssertion
		}

		return handler(ctx, typed)
	})
}

// Dispatch sends a query to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, query Query) (any, error) {
	err := d.ensureOpen("query.dispatch_failed", "dispatching query type "+string(query.Type()))
	if err != nil {
		return nil, err
	}

	wrapped, err := d.inner.Dispatch(string(query.Type()))
	if err != nil {
		if errors.Is(err, dispatcher.ErrHandlerNotFound) {
			return nil, event.WrapRejection(
				ErrHandlerNotFound,
				"query.handler_not_found",
				"no handler registered for query: "+string(query.Type()),
			)
		}

		return nil, event.Wrap(
			err,
			event.Classify(err),
			"query.handler_failed",
			"query type "+string(query.Type()),
		)
	}

	return wrapped(ctx, query)
}

func (d *Dispatcher) ensureOpen(code, msg string) error {
	closedErr := d.inner.CheckClosed(ErrDispatcherClosed)
	if closedErr != nil {
		return event.WrapInfrastructure(closedErr, code, msg)
	}

	return nil
}

// DispatchTyped sends a query and returns a typed result.
func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error) {
	var zero T

	result, err := d.Dispatch(ctx, query)
	if err != nil {
		return zero, err
	}

	typed, ok := result.(T)
	if !ok {
		return zero, event.NewCorruption(
			"query.type_mismatch",
			"unexpected result type for query "+string(query.Type()),
		)
	}

	return typed, nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	closeErr := d.inner.Close()
	if closeErr != nil {
		return event.WrapInfrastructure(closeErr, "query.dispatcher_close",
			"close query dispatcher")
	}

	return nil
}
