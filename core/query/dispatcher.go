// Package query provides query dispatching for CQRS.
package query

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// Handler processes a query and returns a result.
type Handler = func(context.Context, Query) (any, error)

// Dispatcher routes queries to their handlers.
type Dispatcher struct {
	dispatcher.CatalogDispatcher[Type, CatalogMeta]

	base dispatcher.BaseDispatcher[Handler, Middleware]
}

// NewDispatcher creates a new query dispatcher.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{} //nolint:exhaustruct // embedded generic fields require Init method
	d.InitCatalogDispatcher()

	base := dispatcher.NewBaseDispatcher[Handler, Middleware]()
	d.base = base

	return d
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.base.Use(middleware...)
}

// Register binds a handler to a query type.
func (d *Dispatcher) Register(queryType Type, handler Handler) error {
	err := d.base.Lifecycle().CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errors.Wrapf(err, "registering query type %s", queryType)
	}

	err = d.base.Register(string(queryType), handler)
	if err != nil {
		return errors.Wrapf(err, "registering handler for query type %s", queryType)
	}

	return nil
}

// Dispatch sends a query to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, query Query) (any, error) {
	err := d.base.Lifecycle().CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return nil, errors.Wrapf(err, "dispatching query %s", query.Type())
	}

	_, ok := d.base.GetHandler(string(query.Type()))
	if !ok {
		return nil, errors.Wrapf(ErrQueryNotSupported, "query type: %s", query.Type())
	}

	wrapped, err := d.base.Dispatch(
		string(query.Type()),
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "query type: %s", query.Type())
	}

	return wrapped(ctx, query)
}

// DispatchTyped sends a query and returns a typed result.
//
//nolint:ireturn // generic return by design
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
		//nolint:wrapcheck // Newf creates a new error with context
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
	//nolint:wrapcheck // Close returns lifecycle error; caller handles it
	return d.base.Lifecycle().Close()
}
