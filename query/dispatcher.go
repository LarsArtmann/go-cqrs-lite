package query

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// Dispatcher routes queries to their handlers
type Dispatcher struct {
	mu         sync.RWMutex
	handlers   map[Type]func(Query) (any, error)
	middleware []Middleware
	closed     bool
}

// Middleware wraps query handlers for cross-cutting concerns
type Middleware func(func(Query) (any, error)) func(Query) (any, error)

// NewDispatcher creates a new query dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[Type]func(Query) (any, error)),
	}
}

// Use adds middleware to the dispatcher
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middleware = append(d.middleware, middleware...)
}

// Register binds a handler to a query type
func (d *Dispatcher) Register(queryType Type, handler func(Query) (any, error)) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}

	d.handlers[queryType] = handler
	return nil
}

// Dispatch sends a query to its registered handler
func (d *Dispatcher) Dispatch(ctx context.Context, query Query) (any, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, ErrDispatcherClosed
	}

	handler, ok := d.handlers[query.Type()]
	if !ok {
		return nil, errors.Wrapf(ErrQueryNotSupported, "query type: %s", query.Type())
	}

	wrapped := handler
	for i := len(d.middleware) - 1; i >= 0; i-- {
		wrapped = d.middleware[i](wrapped)
	}

	return wrapped(query)
}

// DispatchTyped sends a query and returns a typed result
func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error) {
	var zero T
	result, err := d.Dispatch(ctx, query)
	if err != nil {
		return zero, errors.Wrapf(err, "dispatch failed for query type %q", query.Type())
	}
	typed, ok := result.(T)
	if !ok {
		return zero, errors.Newf("unexpected result type for query %q: got %T", query.Type(), result)
	}
	return typed, nil
}

// Close marks the dispatcher as closed
func (d *Dispatcher) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}
