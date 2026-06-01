// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import (
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

type middlewareChain[H, M any] struct {
	mu         sync.RWMutex
	middleware []M
}

// Add appends middleware to the chain.
func (c *middlewareChain[H, M]) Add(middleware ...M) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.middleware = append(c.middleware, middleware...)
}

// Apply wraps a handler with all middleware in reverse order (last added runs first).
// The wrap function converts a middleware and handler into a wrapped handler.
func (c *middlewareChain[H, M]) Apply(handler H, wrap func(M, H) H) H {
	c.mu.RLock()
	defer c.mu.RUnlock()

	wrapped := handler
	for _, m := range slices.Backward(c.middleware) {
		wrapped = wrap(m, wrapped)
	}

	return wrapped
}

// Middleware returns a copy of the middleware slice for read access.
func (c *middlewareChain[H, M]) Middleware() []M {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return slices.Clone(c.middleware)
}

// Dispatcher is a generic dispatcher that routes requests to their handlers.
type Dispatcher[H any, M any] struct {
	handlers   map[string]H
	handlersMu sync.RWMutex
	Lifecycle  Lifecycle
	middleware middlewareChain[H, M]
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher[H, M any]() *Dispatcher[H, M] {
	return &Dispatcher[H, M]{
		handlers:   make(map[string]H),
		handlersMu: sync.RWMutex{},
		Lifecycle:  Lifecycle{mu: sync.RWMutex{}, closed: false},
		middleware: middlewareChain[H, M]{mu: sync.RWMutex{}, middleware: nil},
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher[H, M]) Use(middleware ...M) {
	d.middleware.Add(middleware...)
}

// Register binds a handler to a type, applying middleware immediately.
// The wrap function converts middleware and handler into a wrapped handler.
// Middleware must be configured via Use() before Register() is called.
func (d *Dispatcher[H, M]) Register(t string, handler H, wrap func(M, H) H) error {
	err := d.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return err
	}

	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	if _, exists := d.handlers[t]; exists {
		return errorfamily.WrapConflict(
			ErrHandlerAlreadyRegistered,
			"dispatcher.handler_registered",
			"handler already registered for type "+t,
		)
	}

	d.handlers[t] = d.middleware.Apply(handler, wrap)

	return nil
}

// getHandler returns the handler for a type and whether it exists.
func (d *Dispatcher[H, M]) getHandler(t string) (H, bool) {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()

	h, ok := d.handlers[t]

	return h, ok
}

// Dispatch returns the wrapped handler for a type.
// The caller is responsible for invoking the returned handler with appropriate arguments.
func (d *Dispatcher[H, M]) Dispatch(t string) (H, error) {
	err := d.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		var zero H

		return zero, err
	}

	h, ok := d.getHandler(t)
	if !ok {
		var zero H

		return zero, errorfamily.WrapRejection(
			ErrHandlerNotFound,
			"dispatcher.handler_not_found",
			"handler not found for type "+t,
		)
	}

	return h, nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher[H, M]) Close() error {
	return d.Lifecycle.Close()
}
