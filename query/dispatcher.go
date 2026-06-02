package query

import (
	"context"
	"errors"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
	err := d.checkClosed("query.register_failed", "registering query type "+string(queryType))
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
		return errorfamily.WrapInfrastructure(
			err,
			"query.register_handler_failed",
			"registering handler for query type "+string(queryType),
		)
	}

	return nil
}

// RegisterTyped binds a typed handler to a query type.
// The handler is wrapped to match the Handler signature, providing
// compile-time type safety for the result type.
func RegisterTyped[T any](d *Dispatcher, queryType Type, handler TypedHandler[T]) error {
	return d.Register(queryType, func(ctx context.Context, q Query) (any, error) {
		return handler(ctx, q)
	})
}

// Dispatch sends a query to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, query Query) (any, error) {
	err := d.checkClosed("query.dispatch_failed", "dispatching query type "+string(query.Type()))
	if err != nil {
		return nil, err
	}

	wrapped, err := d.inner.Dispatch(string(query.Type()))
	if err != nil {
		if errors.Is(err, dispatcher.ErrHandlerNotFound) {
			return nil, errorfamily.WrapRejection(
				ErrHandlerNotFound,
				"query.handler_not_found",
				"no handler registered for query: "+string(query.Type()),
			)
		}

		return nil, errorfamily.Wrap(
			err,
			errorfamily.Classify(err),
			"query.handler_failed",
			"query type "+string(query.Type()),
		)
	}

	return wrapped(ctx, query)
}

func (d *Dispatcher) checkClosed(code, msg string) error {
	err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, code, msg)
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
		return zero, errorfamily.NewCorruption(
			"query.type_mismatch",
			"unexpected result type for query "+string(query.Type()),
		)
	}

	return typed, nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	err := d.inner.Close()
	if err != nil {
		return event.WrapInfrastructure(err, "query.dispatcher_close",
			"close query dispatcher")
	}

	return nil
}
