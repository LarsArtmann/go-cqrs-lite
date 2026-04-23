package memory

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

type MemoryBus struct {
	mu          sync.RWMutex
	handlers    map[event.Type][]event.Handler
	allHandlers []event.Handler
	middleware  []event.Middleware
	closed      bool
}

var _ event.Bus = (*MemoryBus)(nil)

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		mu:          sync.RWMutex{},
		handlers:    make(map[event.Type][]event.Handler),
		allHandlers: nil,
		middleware:  nil,
		closed:      false,
	}
}

func (b *MemoryBus) Use(middleware ...event.Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)
}

func (b *MemoryBus) Publish(ctx context.Context, events ...event.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.Wrap(event.ErrBusClosed, "cannot publish events")
	}

	for i, evt := range events {
		err := b.publishEvent(ctx, evt)
		if err != nil {
			return errors.Wrapf(
				err,
				"failed to publish event %d (%s) from batch of %d events",
				i,
				evt.Type(),
				len(events),
			)
		}
	}

	return nil
}

func (b *MemoryBus) publishEvent(ctx context.Context, evt event.Event) error {
	handler := func(ctx context.Context, e event.Event) error {
		err := b.notifyHandlers(ctx, e, b.allHandlers, "all-handler")
		if err != nil {
			return err
		}

		err = b.notifyHandlers(ctx, e, b.handlers[e.Type()], "handler")
		if err != nil {
			return err
		}

		return nil
	}

	for i := len(b.middleware) - 1; i >= 0; i-- {
		handler = b.middleware[i](handler)
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
			return errors.Wrapf(err, "%s %d failed for event %s", prefix, idx, evt.Type())
		}
	}

	return nil
}

func (b *MemoryBus) Subscribe(eventType event.Type, handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.ErrBusClosed
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}

func (b *MemoryBus) SubscribeAll(handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.ErrBusClosed
	}

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

func (b *MemoryBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true

	return nil
}
