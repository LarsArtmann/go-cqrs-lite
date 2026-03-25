package event

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// MemoryBus is an in-memory implementation of Bus for testing and development
type MemoryBus struct {
	mu          sync.RWMutex
	handlers    map[EventType][]Handler
	allHandlers []Handler
	middleware  []Middleware
	closed      bool
}

// NewMemoryBus creates a new in-memory event bus
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers: make(map[EventType][]Handler),
	}
}

// Use adds middleware to the bus
func (b *MemoryBus) Use(middleware ...Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, middleware...)
}

// Publish sends events to all registered handlers
func (b *MemoryBus) Publish(ctx context.Context, events ...Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.Wrap(ErrBusClosed, "cannot publish events")
	}

	for i, event := range events {
		if err := b.publishEvent(ctx, event); err != nil {
			return errors.Wrapf(err, "failed to publish event %d (%s) from batch of %d events", i, event.Type(), len(events))
		}
	}
	return nil
}

func (b *MemoryBus) publishEvent(ctx context.Context, event Event) error {
	handler := func(ctx context.Context, e Event) error {
		for i, h := range b.allHandlers {
			if err := h(ctx, e); err != nil {
				return errors.Wrapf(err, "all-handler %d failed for event %s", i, e.Type())
			}
		}

		handlers := b.handlers[e.Type()]
		for i, h := range handlers {
			if err := h(ctx, e); err != nil {
				return errors.Wrapf(err, "handler %d failed for event %s", i, e.Type())
			}
		}
		return nil
	}

	for i := len(b.middleware) - 1; i >= 0; i-- {
		handler = b.middleware[i](handler)
	}

	return handler(ctx, event)
}

// Subscribe registers a handler for specific event types
func (b *MemoryBus) Subscribe(eventType EventType, handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBusClosed
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

// SubscribeAll registers a handler for all event types
func (b *MemoryBus) SubscribeAll(handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBusClosed
	}

	b.allHandlers = append(b.allHandlers, handler)
	return nil
}

// Close marks the bus as closed
func (b *MemoryBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}
