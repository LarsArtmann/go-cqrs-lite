package dispatcher

import "sync"

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
