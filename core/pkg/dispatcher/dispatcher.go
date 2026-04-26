// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import (
	"errors"
	"fmt"
	"maps"
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
// It embeds LifecycleMixin for reusable closed state management.
type Lifecycle struct {
	LifecycleMixin
}

// ErrHandlerNotFound is returned when no handler is registered for a type.
var ErrHandlerNotFound = errors.New("handler not found")

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errors.New("dispatcher is closed")

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
//nolint:ireturn // generic interface return by design
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
	handlers   map[string]H
	handlersMu sync.RWMutex
	Lifecycle  Lifecycle
	Middleware MiddlewareChain[H, M]
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher[H, M any]() *Dispatcher[H, M] {
	return &Dispatcher[H, M]{
		handlers: make(map[string]H),
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
	err := d.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return fmt.Errorf("dispatcher is closed: %w", err)
	}

	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	d.handlers[t] = handler

	return nil
}

// GetHandler returns the handler for a type and whether it exists.
//nolint:ireturn // generic interface return by design
func (d *Dispatcher[H, M]) GetHandler(t string) (H, bool) {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()

	h, ok := d.handlers[t]

	return h, ok
}

// Dispatch sends a request to its registered handler and returns the wrapped handler.
// The caller is responsible for invoking the wrapped handler with appropriate arguments.
//nolint:ireturn // generic interface return by design
func (d *Dispatcher[H, M]) Dispatch(t string, wrap func(M, H) H) (H, error) {
	err := d.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		var zero H

		return zero, fmt.Errorf("dispatcher is closed: %w", err)
	}

	h, ok := d.GetHandler(t)
	if !ok {
		var zero H

		//nolint:err113 // dynamic error required to include type name for debugging
		return zero, fmt.Errorf("handler not found for type: %s", t)
	}

	wrapped := d.Middleware.Apply(h, wrap)

	return wrapped, nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher[H, M]) Close() error {
	return d.Lifecycle.Close()
}

// CopyCatalogEntries copies entries from src to dest and returns dest.
func CopyCatalogEntries[KT comparable, VT any](dest, src map[KT]VT) map[KT]VT {
	if dest == nil {
		dest = make(map[KT]VT, len(src))
	}

	maps.Copy(dest, src)

	return dest
}

// CatalogDispatcher is a mixin that provides catalog entry management.
// KT is the type key (e.g., command.Type or query.Type).
// VT is the catalog metadata type (e.g., command.CatalogMeta or query.CatalogMeta).
type CatalogDispatcher[KT comparable, VT any] struct {
	catalogEntries map[KT]VT
}

// InitCatalogDispatcher initializes the catalog entries map.
func (c *CatalogDispatcher[KT, VT]) InitCatalogDispatcher() {
	c.catalogEntries = make(map[KT]VT)
}

// RegisterCatalogEntry stores catalog metadata for a type.
// This is a side channel that doesn't affect dispatch behavior.
func (c *CatalogDispatcher[KT, VT]) RegisterCatalogEntry(key KT, meta VT) {
	c.catalogEntries[key] = meta
}

// CatalogEntries returns a copy of all registered catalog entries.
func (c *CatalogDispatcher[KT, VT]) CatalogEntries() map[KT]VT {
	return CopyCatalogEntries(nil, c.catalogEntries)
}
