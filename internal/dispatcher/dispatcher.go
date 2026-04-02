// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import (
	"errors"
	"fmt"
	"sync"
)

// LifecycleMixin provides thread-safe closed state management for composable types.
type LifecycleMixin struct {
	mu     sync.RWMutex
	closed bool
}

// Close marks the lifecycle as closed. It is safe to call multiple times.
func (m *LifecycleMixin) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// IsClosed returns true if the lifecycle has been closed.
func (m *LifecycleMixin) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// CheckClosed returns an error if the lifecycle is closed, or nil otherwise.
func (m *LifecycleMixin) CheckClosed(closedErr error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return closedErr
	}
	return nil
}

// Lifecycle manages the closed state of a dispatcher with thread-safe access.
type Lifecycle struct {
	LifecycleMixin
}

// Close marks the lifecycle as closed. It is safe to call multiple times.
func (l *Lifecycle) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

// IsClosed returns true if the lifecycle has been closed.
func (l *Lifecycle) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

// CheckClosed returns an error if the lifecycle is closed, or nil otherwise.
func (l *Lifecycle) CheckClosed(closedErr error) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return closedErr
	}
	return nil
}

// MiddlewareChain stores and applies middleware in a thread-safe manner.
// H is the handler type, and M is the middleware type that wraps handlers.
type MiddlewareChain[H, M any] struct {
	mu         sync.RWMutex
	middleware []M
}

// Add appends middleware to the chain.
func (c *MiddlewareChain[H, M]) Add(middleware ...M) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middleware = append(c.middleware, middleware...)
}

// Apply wraps a handler with all middleware in reverse order (last added runs first).
// The wrap function converts a middleware and handler into a wrapped handler.
func (c *MiddlewareChain[H, M]) Apply(handler H, wrap func(M, H) H) H {
	c.mu.RLock()
	defer c.mu.RUnlock()

	wrapped := handler
	for i := len(c.middleware) - 1; i >= 0; i-- {
		wrapped = wrap(c.middleware[i], wrapped)
	}
	return wrapped
}

// Middleware returns the middleware slice for read access.
func (c *MiddlewareChain[H, M]) Middleware() []M {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.middleware
}

// Dispatcher is a generic dispatcher that routes requests to their handlers.
type Dispatcher[H any, M any] struct {
	Handlers   map[string]H
	Lifecycle  Lifecycle
	Middleware MiddlewareChain[H, M]
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher[H, M any]() *Dispatcher[H, M] {
	return &Dispatcher[H, M]{
		Handlers: make(map[string]H),
		Lifecycle: Lifecycle{
			LifecycleMixin: LifecycleMixin{
				mu:     sync.RWMutex{},
				closed: false,
			},
		},
		Middleware: MiddlewareChain[H, M]{
			mu:         sync.RWMutex{},
			middleware: nil,
		},
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher[H, M]) Use(middleware ...M) {
	d.Middleware.Add(middleware...)
}

// Register binds a handler to a type.
func (d *Dispatcher[H, M]) Register(t string, handler H) error {
	if err := d.Lifecycle.CheckClosed(ErrHandlerNotFound); err != nil {
		return fmt.Errorf("dispatcher is closed: %w", err)
	}
	d.Handlers[t] = handler
	return nil
}

// Dispatch sends a request to its registered handler and returns the wrapped handler.
// The caller is responsible for invoking the wrapped handler with appropriate arguments.
func (d *Dispatcher[H, M]) Dispatch(t string, _ H, wrap func(M, H) H) (H, error) {
	if err := d.Lifecycle.CheckClosed(ErrHandlerNotFound); err != nil {
		var zero H
		return zero, fmt.Errorf("dispatcher is closed: %w", err)
	}

	h, ok := d.Handlers[t]
	if !ok {
		var zero H
		return zero, fmt.Errorf("handler not found for type: %s", t)
	}

	wrapped := d.Middleware.Apply(h, wrap)
	return wrapped, nil
}

// ErrHandlerNotFound is returned when no handler is registered for a type.
var ErrHandlerNotFound = errors.New("handler not found")

// Close marks the dispatcher as closed.
func (d *Dispatcher[H, M]) Close() error {
	return d.Lifecycle.Close()
}
