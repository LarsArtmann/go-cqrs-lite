package projection

import (
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// HandlerRegistry maps event types to handler functions.
// Thread-safe. Call On() to register handlers before starting the Runner.
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[event.Type][]event.Handler
	wildcard []event.Handler
}

// NewHandlerRegistry creates an empty handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[event.Type][]event.Handler),
	}
}

// On registers a handler for a specific event type.
// Returns an error if the handler is nil.
func (r *HandlerRegistry) On(eventType event.Type, handler event.Handler) error {
	if handler == nil {
		return ErrNilHandler
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[eventType] = append(r.handlers[eventType], handler)

	return nil
}

// OnAll registers a handler for all event types (wildcard).
func (r *HandlerRegistry) OnAll(handler event.Handler) error {
	if handler == nil {
		return ErrNilHandler
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.wildcard = append(r.wildcard, handler)

	return nil
}

// Lookup returns handlers for the given event type (specific + wildcard).
func (r *HandlerRegistry) Lookup(eventType event.Type) []event.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specific := r.handlers[eventType]

	result := make([]event.Handler, 0, len(specific)+len(r.wildcard))
	result = append(result, specific...)
	result = append(result, r.wildcard...)

	return result
}

// EventTypes returns all registered event types.
func (r *HandlerRegistry) EventTypes() []event.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]event.Type, 0, len(r.handlers))

	for t := range r.handlers {
		types = append(types, t)
	}

	return types
}

// HasHandlers returns true if any handlers are registered.
func (r *HandlerRegistry) HasHandlers() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.handlers) > 0 || len(r.wildcard) > 0
}
