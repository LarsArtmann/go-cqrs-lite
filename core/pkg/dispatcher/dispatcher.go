// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import (
	"maps"
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// CatalogEntry holds basic metadata for a catalog item.
// Used by command and query dispatchers for per-type documentation.
type CatalogEntry struct {
	Name    string
	Version string
	Summary string
}

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

// middlewareChain stores and applies middleware in a thread-safe manner.
// H is the handler type, and M is the middleware type that wraps handlers.
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

	result := make([]M, len(c.middleware))
	copy(result, c.middleware)

	return result
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

// copyCatalogEntries copies entries from src to dest and returns dest.
func copyCatalogEntries[KT comparable, VT any](dest, src map[KT]VT) map[KT]VT {
	if dest == nil {
		dest = make(map[KT]VT, len(src))
	}

	maps.Copy(dest, src)

	return dest
}

// CatalogDispatcher is a mixin that provides catalog entry management.
// KT is the type key (e.g., command.Type or query.Type).
// VT is the catalog metadata type (e.g., dispatcher.CatalogEntry).
type CatalogDispatcher[KT comparable, VT any] struct {
	catalogEntries map[KT]VT
}

// InitCatalogDispatcher initializes the catalog entries map.
func (c *CatalogDispatcher[KT, VT]) InitCatalogDispatcher() {
	c.catalogEntries = make(map[KT]VT)
}

// newCatalogDispatcher creates a new initialized CatalogDispatcher.
func newCatalogDispatcher[KT comparable, VT any]() CatalogDispatcher[KT, VT] {
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
	return copyCatalogEntries(nil, c.catalogEntries)
}
