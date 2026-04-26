package memory

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

type MemoryBus struct {
	dispatcher.LifecycleMixin

	mu          sync.RWMutex
	handlers    map[event.Type][]event.Handler
	allHandlers []event.Handler
	middleware  []event.Middleware
}

var _ event.Bus = (*MemoryBus)(nil)

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		LifecycleMixin: dispatcher.LifecycleMixin{},
		handlers:       make(map[event.Type][]event.Handler),
	}
}

func (b *MemoryBus) Use(middleware ...event.Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)
}

func (b *MemoryBus) Publish(ctx context.Context, events ...event.Event) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

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
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}

func (b *MemoryBus) SubscribeAll(handler event.Handler) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

func (b *MemoryBus) Close() error {
	return b.LifecycleMixin.Close()
}
