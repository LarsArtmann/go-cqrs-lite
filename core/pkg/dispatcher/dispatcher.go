// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Lifecycle provides thread-safe closed state management for composable types.
type Lifecycle struct {
	mu     sync.RWMutex
	closed bool
}

// Close marks the lifecycle as closed. It is safe to call multiple times.
func (m *Lifecycle) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true

	return nil
}

// IsClosed returns true if the lifecycle has been closed.
func (m *Lifecycle) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.closed
}

// CheckClosed returns an error if the lifecycle is closed, or nil otherwise.
func (m *Lifecycle) CheckClosed(closedErr error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return closedErr
	}

	return nil
}

// ErrHandlerNotFound is returned when no handler is registered for a type.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"dispatcher.handler_not_found",
	"handler not found",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"dispatcher.dispatcher_closed",
	"dispatcher is closed",
)

// ErrHandlerAlreadyRegistered is returned when a handler is already registered for a type.
var ErrHandlerAlreadyRegistered = errorfamily.NewConflict(
	"dispatcher.handler_already_registered",
	"handler already registered for type",
)

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
	for _, m := range slices.Backward(c.middleware) {
		wrapped = wrap(m, wrapped)
	}

	return wrapped
}

// Middleware returns a copy of the middleware slice for read access.
func (c *MiddlewareChain[H, M]) Middleware() []M {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]M, len(c.middleware))
	copy(result, c.middleware)

	return result
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
		handlers:   make(map[string]H),
		handlersMu: sync.RWMutex{},
		Lifecycle:  Lifecycle{mu: sync.RWMutex{}, closed: false},
		Middleware: MiddlewareChain[H, M]{mu: sync.RWMutex{}, middleware: nil},
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher[H, M]) Use(middleware ...M) {
	d.Middleware.Add(middleware...)
}

// Register binds a handler to a type, applying middleware immediately.
// The wrap function converts middleware and handler into a wrapped handler.
// Middleware must be configured via Use() before Register() is called.
func (d *Dispatcher[H, M]) Register(t string, handler H, wrap func(M, H) H) error {
	if err := d.Lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return err
	}

	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	if _, exists := d.handlers[t]; exists {
		return errorfamily.WrapConflict(
			ErrHandlerAlreadyRegistered,
			"dispatcher.handler_registered",
			fmt.Sprintf("handler already registered for type %s", t),
		)
	}

	d.handlers[t] = d.Middleware.Apply(handler, wrap)

	return nil
}

// GetHandler returns the handler for a type and whether it exists.
func (d *Dispatcher[H, M]) GetHandler(t string) (H, bool) {
	d.handlersMu.RLock()
	defer d.handlersMu.RUnlock()

	h, ok := d.handlers[t]

	return h, ok
}

// Dispatch returns the wrapped handler for a type.
// The caller is responsible for invoking the returned handler with appropriate arguments.
func (d *Dispatcher[H, M]) Dispatch(t string) (H, error) {
	if err := d.Lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		var zero H

		return zero, err
	}

	h, ok := d.GetHandler(t)
	if !ok {
		var zero H

		return zero, errorfamily.WrapRejection(
			ErrHandlerNotFound,
			"dispatcher.handler_not_found",
			fmt.Sprintf("handler not found for type %s", t),
		)
	}

	return h, nil
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

// NewCatalogDispatcher creates a new initialized CatalogDispatcher.
func NewCatalogDispatcher[KT comparable, VT any]() CatalogDispatcher[KT, VT] {
	c := CatalogDispatcher[KT, VT]{} //nolint:exhaustruct // unexported field requires Init method
	c.InitCatalogDispatcher()

	return c
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
