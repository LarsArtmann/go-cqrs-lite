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
	//nolint:exhaustruct // embedded LifecycleMixin has unexported fields from different package
	return &MemoryBus{
		handlers: make(map[event.Type][]event.Handler),
	}
}

func (b *MemoryBus) Use(middleware ...event.Middleware) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return errors.Wrap(err, "bus use middleware")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)

	return nil
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
		return errors.Wrap(err, "bus publish")
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
		return errors.Wrap(err, "bus subscribe")
	}

	if handler == nil {
		return errors.New("handler must not be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}

func (b *MemoryBus) SubscribeAll(handler event.Handler) error {
	err := b.CheckClosed(event.ErrBusClosed)
	if err != nil {
		return errors.Wrap(err, "bus subscribe all")
	}

	if handler == nil {
		return errors.New("handler must not be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

func (b *MemoryBus) Close() error {
	return b.LifecycleMixin.Close() //nolint:wrapcheck
}
