package storage

import (
	"context"
	"encoding/json/v2"
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// Publish dispatches events to local handlers and sends a NOTIFY so other
// processes can re-fetch and process the event. The NOTIFY payload is a
// lightweight JSON reference — never the full event payload.
//cqrs-lint:ignore(A020) library code or intentional pattern
func (b *PostgresBus) Publish(ctx context.Context, events ...event.Event) error {
	if b.closed.Load() {
		return errorfamily.WrapInfrastructure(event.ErrBusClosed, "storage.pg_bus_publish",
			"postgres bus publish: bus is closed")
	}

	if len(events) == 0 {
		return nil
	}

	return b.cachedPublisher.Publish(ctx, events...)
}

func (b *PostgresBus) rebuildPublisherChain() {
	var inner event.Publisher = event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
		for _, evt := range events {
			if err := b.publishOne(ctx, evt); err != nil {
				return err
			}
		}

		return nil
	})

	for _, m := range slices.Backward(b.publishMiddleware) {
		inner = m(inner)
	}

	b.cachedPublisher = inner
}

func (b *PostgresBus) publishOne(ctx context.Context, evt event.Event) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "pg_bus.publish",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString("cqrs.event.type", string(evt.Type())),
			cqrsotel.AttrString("cqrs.event.id", evt.ID().String()),
		),
	)
	defer span.End()

	if err := b.dispatchLocal(ctx, evt); err != nil {
		cqrsotel.RecordError(span, err)
		return err
	}

	payload := notifyPayload{
		EventID:    evt.ID(),
		EventType:  evt.Type(),
		StreamType: evt.StreamType(),
		StreamID:   evt.StreamID(),
		Version:    evt.Version(),
	}

	payloadJSON, err := json.Marshal(payload, json.Deterministic(true))
	if err != nil {
		cqrsotel.RecordError(span, err)
		return errorfamily.WrapInfrastructure(err, "storage.pg_bus_marshal",
			"marshal notify payload for "+string(evt.Type()))
	}

	err = b.opts.notifyFn(ctx, b.opts.channel, string(payloadJSON))
	if err != nil {
		cqrsotel.RecordError(span, err)
		return errorfamily.WrapInfrastructure(err, "storage.pg_bus_notify",
			"send NOTIFY for "+string(evt.Type()))
	}

	return nil
}

// dispatchLocal sends the event to all matching local handlers via the middleware chain.
func (b *PostgresBus) dispatchLocal(ctx context.Context, evt event.Event) error {
	b.mu.RLock()
	handler := b.cachedHandler
	b.mu.RUnlock()

	return handler(ctx, evt)
}

func (b *PostgresBus) rebuildHandlerChain() {
	inner := event.Handler(func(ctx context.Context, e event.Event) error {
		b.mu.RLock()
		allHandlers := b.allHandlers
		typeHandlers := b.handlers[e.Type()]
		b.mu.RUnlock()

		for _, h := range allHandlers {
			if err := h(ctx, e); err != nil {
				return err
			}
		}

		for _, h := range typeHandlers {
			if err := h(ctx, e); err != nil {
				return err
			}
		}

		return nil
	})

	for _, m := range slices.Backward(b.middleware) {
		inner = m(inner)
	}

	b.cachedHandler = inner
}

// Subscribe registers a handler for a specific event type.
func (b *PostgresBus) Subscribe(eventType event.Type, handler event.Handler) error {
	if handler == nil {
		return errorfamily.WrapInfrastructure(errNilBusHandler, "storage.pg_bus_subscribe",
			"subscribe: nil handler")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.rebuildHandlerChain()

	return nil
}

// SubscribeAll registers a catch-all handler that receives every event.
func (b *PostgresBus) SubscribeAll(handler event.Handler) error {
	if handler == nil {
		return errorfamily.WrapInfrastructure(errNilBusHandler, "storage.pg_bus_subscribe_all",
			"subscribe all: nil handler")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)
	b.rebuildHandlerChain()

	return nil
}

// appendMiddleware is the shared Use/UsePublish body: lock, append, rebuild.
// Uses sync.Locker so it works with both sync.Mutex and sync.RWMutex.
//
// watermill/bus_helpers.go has a functionally equivalent helper
// (withLockedModify) that uses a closure-based signature instead of a
// pointer-to-slice. The two implementations are deliberately structurally
// different to avoid an accidental cross-module clone between these
// independent transport adapters.
func appendMiddleware[M any](mu sync.Locker, middleware *[]M, mw []M, rebuild func()) {
	mu.Lock()
	defer mu.Unlock()
	*middleware = append(*middleware, mw...)
	rebuild()
}

// Use adds middleware that wraps all event handlers.
func (b *PostgresBus) Use(mw ...event.Middleware) error {
	appendMiddleware(&b.mu, &b.middleware, mw, b.rebuildHandlerChain)
	return nil
}

// UsePublish adds middleware that wraps the Publish path.
func (b *PostgresBus) UsePublish(mw ...event.PublishMiddleware) error {
	appendMiddleware(&b.mu, &b.publishMiddleware, mw, b.rebuildPublisherChain)
	return nil
}
