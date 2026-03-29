// Package query provides query dispatching for CQRS.
package query

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/internal/dispatcher"
)

// Handler processes a query and returns a result.
type Handler = func(Query) (any, error)

// Dispatcher routes queries to their handlers.
type Dispatcher struct {
	handlers   map[Type]Handler
	lifecycle  dispatcher.Lifecycle
	middleware dispatcher.MiddlewareChain[Handler, Middleware]
}

// NewDispatcher creates a new query dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers:   make(map[Type]Handler),
		lifecycle:  dispatcher.Lifecycle{},
		middleware: dispatcher.MiddlewareChain[Handler, Middleware]{},
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.middleware.Add(middleware...)
}

// Register binds a handler to a query type.
func (d *Dispatcher) Register(queryType Type, handler Handler) error {
	if err := d.lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return errors.Wrapf(err, "registering query type %s", queryType)
	}
	d.handlers[queryType] = handler
	return nil
}

// Dispatch sends a query to its registered handler.
func (d *Dispatcher) Dispatch(_ context.Context, query Query) (any, error) {
	if err := d.lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return nil, errors.Wrapf(err, "dispatching query %s", query.Type())
	}

	handler, ok := d.handlers[query.Type()]
	if !ok {
		return nil, errors.Wrapf(ErrQueryNotSupported, "query type: %s", query.Type())
	}

	wrapped := d.middleware.Apply(handler, func(m Middleware, h Handler) Handler {
		return m(h)
	})

	return wrapped(query)
}

// DispatchTyped sends a query and returns a typed result.
func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error) {
	var zero T
	result, err := d.Dispatch(ctx, query)
	if err != nil {
		return zero, errors.Wrapf(
			err,
			"dispatch failed for query type %q (expected type: %T)",
			query.Type(),
			zero,
		)
	}
	typed, ok := result.(T)
	if !ok {
		return zero, errors.Newf(
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
	return d.lifecycle.Close()
}
