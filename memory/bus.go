package memory

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// MemoryBus is an in-memory implementation of event.Bus for testing and single-process deployments.
// It is safe for concurrent use. Handler execution blocks publishers (see Publish docs).
type MemoryBus struct {
	dispatcher.Lifecycle

	mu                sync.RWMutex
	handlers          map[event.Type][]event.Handler
	allHandlers       []event.Handler
	middleware        []event.Middleware
	publishMiddleware []event.PublishMiddleware
}

var (
	_ event.Bus = (*MemoryBus)(nil)
	_ io.Closer = (*MemoryBus)(nil)
)

// NewMemoryBus creates a new in-memory event bus.
func NewMemoryBus() *MemoryBus {
	//nolint:exhaustruct // embedded Lifecycle has unexported fields from different package
	return &MemoryBus{
		handlers: make(map[event.Type][]event.Handler),
	}
}

func (b *MemoryBus) useLocked(name string, fn func()) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.bus_operation_failed", "bus "+name+" failed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	fn()

	return nil
}

// Use registers event middleware. Middleware is applied in reverse registration order
// (last registered runs first). Returns ErrBusClosed if the bus is already closed.
func (b *MemoryBus) Use(mw ...event.Middleware) error {
	return b.useLocked("use middleware", func() {
		b.middleware = append(b.middleware, mw...)
	})
}

// UsePublish registers publish-side middleware. Returns ErrBusClosed if the bus is already closed.
func (b *MemoryBus) UsePublish(mw ...event.PublishMiddleware) error {
	return b.useLocked("use publish middleware", func() {
		b.publishMiddleware = append(b.publishMiddleware, mw...)
	})
}

// Publish sends events to all matching subscribers.
//
// The MemoryBus is designed for testing and single-process deployments.
//
// Ordering: Within a single event, all SubscribeAll handlers run before
// type-specific handlers. If any handler fails, subsequent handlers for
// that event are skipped.
//
// Partial publish: Publish sends events sequentially. If event N fails to
// publish, events 0..N-1 have already been delivered. There is no rollback.
// This mirrors real-world at-least-once delivery semantics.
//
// Concurrency: Publish holds a read lock for the duration of handler
// execution — subscribers block publishers until all handlers complete.
// This is acceptable for test utilities but limits throughput.
func (b *MemoryBus) Publish(ctx context.Context, events ...event.Event) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.publish_failed", "bus publish")
	}

	b.mu.RLock()
	publishMw := b.publishMiddleware
	b.mu.RUnlock()

	inner := event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
		b.mu.RLock()
		defer b.mu.RUnlock()

		for i, evt := range events {
			err := b.publishEvent(ctx, evt)
			if err != nil {
				return event.Wrapf(
					err,
					event.Infrastructure,
					"memory.publish_event_failed",
					"failed to publish event %d (%s) from batch of %d events",
					i,
					evt.Type(),
					len(events),
				)
			}
		}

		return nil
	})

	publisher := event.Publisher(inner)
	for _, m := range slices.Backward(publishMw) {
		publisher = m(publisher)
	}

	err = publisher.Publish(ctx, events...)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.publish_failed", "memory bus publish")
	}

	return nil
}

func (b *MemoryBus) publishEvent(ctx context.Context, evt event.Event) error {
	handler := func(ctx context.Context, e event.Event) error {
		err := b.notifyHandlers(ctx, e, b.allHandlers, "all-handler")
		if err != nil {
			return event.WrapInfrastructure(err, "memory.notify_all_handlers_failed", "notify all-handlers for "+string(e.Type()))
		}

		err = b.notifyHandlers(ctx, e, b.handlers[e.Type()], "handler")
		if err != nil {
			return event.WrapInfrastructure(err, "memory.notify_handler_failed", "notify handler for "+string(e.Type()))
		}

		return nil
	}

	for _, m := range slices.Backward(b.middleware) {
		handler = m(handler)
	}

	return handler(ctx, evt)
}

func (b *MemoryBus) notifyHandlers(
	ctx context.Context,
	evt event.Event,
	handlers []event.Handler,
	prefix string,
) error {
	for idx, h := range handlers {
		err := h(ctx, evt)
		if err != nil {
			return event.Wrapf(
				err,
				event.Infrastructure,
				"memory.handler_failed",
				"%s %d failed for event %s",
				prefix,
				idx,
				evt.Type(),
			)
		}
	}

	return nil
}

func (b *MemoryBus) register(name string, handler event.Handler, fn func()) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.bus_register_failed", "bus "+name+" failed")
	}

	if handler == nil {
		return ErrHandlerNil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	fn()

	return nil
}

// Subscribe registers a handler for a specific event type. Returns ErrHandlerNil if
// the handler is nil, or ErrBusClosed if the bus is closed.
func (b *MemoryBus) Subscribe(eventType event.Type, handler event.Handler) error {
	return b.register("subscribe", handler, func() {
		b.handlers[eventType] = append(b.handlers[eventType], handler)
	})
}

// SubscribeAll registers a handler that receives every published event regardless of type.
// All-handlers run before type-specific handlers (see Publish docs).
func (b *MemoryBus) SubscribeAll(handler event.Handler) error {
	return b.register("subscribe all", handler, func() {
		b.allHandlers = append(b.allHandlers, handler)
	})
}

// Close marks the bus as closed. Subsequent Publish, Subscribe, or Use calls return ErrBusClosed.
func (b *MemoryBus) Close() error {
	return b.Lifecycle.Close() //nolint:wrapcheck
}
