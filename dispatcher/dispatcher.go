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

// Apply wraps a handler with all middleware. The first-added middleware
// becomes the outermost wrapper (runs first in the execution chain).
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

// handlerEntry stores a raw handler and its wrap function so middleware
// can be applied at dispatch time rather than registration time.
type handlerEntry[H, M any] struct {
	handler H
	wrap    func(M, H) H
}

// Dispatcher is a generic dispatcher that routes requests to their handlers.
type Dispatcher[H any, M any] struct {
	handlers   map[string]handlerEntry[H, M]
	handlersMu sync.RWMutex
	lifecycle  Lifecycle
	middleware middlewareChain[H, M]
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher[H, M any]() *Dispatcher[H, M] {
	return &Dispatcher[H, M]{
		handlers:   make(map[string]handlerEntry[H, M]),
		handlersMu: sync.RWMutex{},
		lifecycle:  Lifecycle{mu: sync.RWMutex{}, closed: false},
		middleware: middlewareChain[H, M]{mu: sync.RWMutex{}, middleware: nil},
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher[H, M]) Use(middleware ...M) {
	d.middleware.Add(middleware...)
}

// Register binds a handler to a type.
// The wrap function converts middleware and handler into a wrapped handler.
// Middleware is applied at dispatch time, so Use() can be called in any order
// relative to Register().
func (d *Dispatcher[H, M]) Register(t string, handler H, wrap func(M, H) H) error {
	err := d.lifecycle.CheckClosed(ErrDispatcherClosed)
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

	d.handlers[t] = handlerEntry[H, M]{handler: handler, wrap: wrap}

	return nil
}

// getHandler returns the middleware-wrapped handler for a type and whether it exists.
// Middleware is applied at dispatch time using the current chain.
func (d *Dispatcher[H, M]) getHandler(t string) (H, bool) {
	d.handlersMu.RLock()
	entry, ok := d.handlers[t]
	d.handlersMu.RUnlock()

	if !ok {
		var zero H

		return zero, false
	}

	return d.middleware.Apply(entry.handler, entry.wrap), true
}

// Dispatch returns the wrapped handler for a type.
// The caller is responsible for invoking the returned handler with appropriate arguments.
func (d *Dispatcher[H, M]) Dispatch(t string) (H, error) {
	err := d.lifecycle.CheckClosed(ErrDispatcherClosed)
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
	return d.lifecycle.Close()
}

// IsClosed reports whether the dispatcher has been closed.
func (d *Dispatcher[H, M]) IsClosed() bool {
	return d.lifecycle.IsClosed()
}

// CheckClosed returns the provided error if the dispatcher is closed.
func (d *Dispatcher[H, M]) CheckClosed(closedErr error) error {
	return d.lifecycle.CheckClosed(closedErr)
}
