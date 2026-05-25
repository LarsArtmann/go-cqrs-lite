// Package query provides query dispatching for CQRS.
package query

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
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
	dispatcher.CatalogDispatcher[Type, dispatcher.HandlerMeta]

	inner *dispatcher.Dispatcher[Handler, Middleware]
}

var _ io.Closer = (*Dispatcher)(nil)

// NewDispatcher creates a new query dispatcher.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{} //nolint:exhaustruct // embedded generic fields require Init method
	d.InitCatalogDispatcher()
	d.inner = dispatcher.NewDispatcher[Handler, Middleware]()

	return d
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.inner.Use(middleware...)
}

// Register binds a handler to a query type.
func (d *Dispatcher) Register(queryType Type, handler Handler) error {
	err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return fmt.Errorf("registering query type %s: %w", queryType, err)
	}

	err = d.inner.Register(
		string(queryType),
		handler,
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return fmt.Errorf("registering handler for query type %s: %w", queryType, err)
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
	wrapped, err := d.inner.Dispatch(string(query.Type()))
	if err != nil {
		if errors.Is(err, dispatcher.ErrHandlerNotFound) {
			return nil, fmt.Errorf("%w: query type: %s", ErrQueryNotSupported, query.Type())
		}

		return nil, fmt.Errorf("query type %s: %w", query.Type(), err)
	}

	return wrapped(ctx, query)
}

// DispatchTyped sends a query and returns a typed result.
func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error) {
	var zero T

	result, err := d.Dispatch(ctx, query)
	if err != nil {
		return zero, fmt.Errorf(
			"dispatch failed for query type %q (expected type: %T): %w",
			query.Type(),
			zero,
			err,
		)
	}

	typed, ok := result.(T)
	if !ok {
		//nolint:err113 // dynamic error with runtime type info; no useful sentinel
		return zero, fmt.Errorf(
			"unexpected result type for query %q: got %T, expected: %T",
			query.Type(),
			result,
			zero,
		)
	}

	return typed, nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	//nolint:wrapcheck // Close returns lifecycle error; caller handles it
	return d.inner.Close()
}
