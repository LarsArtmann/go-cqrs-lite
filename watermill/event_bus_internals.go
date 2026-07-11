package watermill

import (
	"context"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func (b *EventBus) rebuildPublisherChain() {
	var inner event.Publisher = event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		for _, evt := range events {
			msg := eventToMessage(evt)
			if err := b.publisher.Publish(b.topic, msg); err != nil {
				return errorfamily.WrapInfrastructure(err, "watermill.event_bus_publish",
					"publish to topic "+b.topic)
			}
		}

		return nil
	})

	for _, v := range slices.Backward(b.publishMiddleware) {
		inner = v(inner)
	}

	b.cachedPublisher = inner
}

func (b *EventBus) rebuildHandlerChain() {
	allHandlers := make([]event.Handler, len(b.allHandlers))
	copy(allHandlers, b.allHandlers)

	typeSnapshot := make(map[event.Type][]event.Handler, len(b.typeHandlers))
	for k, v := range b.typeHandlers {
		cp := make([]event.Handler, len(v))
		copy(cp, v)
		typeSnapshot[k] = cp
	}

	inner := event.Handler(func(ctx context.Context, evt event.Event) error {
		for _, h := range allHandlers {
			if err := h(ctx, evt); err != nil {
				return err
			}
		}

		for _, h := range typeSnapshot[evt.Type()] {
			if err := h(ctx, evt); err != nil {
				return err
			}
		}

		return nil
	})

	for _, v := range slices.Backward(b.middleware) {
		inner = v(inner)
	}

	b.cachedHandler = inner
}

func (b *EventBus) dispatchLocal(ctx context.Context, evt event.Event) error {
	b.mu.Lock()
	handler := b.cachedHandler
	b.mu.Unlock()

	return handler(ctx, evt)
}

func (b *EventBus) ensureSubscriptionLocked() {
	if b.subStarted {
		return
	}

	b.subCtx, b.subCancel = context.WithCancel(context.Background())

	msgs, err := b.subscriber.Subscribe(b.subCtx, b.topic)
	if err != nil {
		b.logger.ErrorContext(b.subCtx, "watermill: subscribe failed",
			"error", err, "topic", b.topic)
		b.subCancel()
		b.subCtx = nil
		b.subCancel = nil

		return
	}

	b.subStarted = true

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				evt, decodeErr := MessageToEvent(b.topic, msg)
				if decodeErr != nil {
					// Decode failure is non-transient (same bytes → same error).
					// Ack to prevent infinite retry loops, especially under
					// BlockPublishUntilSubscriberAck which would deadlock.
					b.logger.ErrorContext(b.subCtx, "watermill: decode message failed",
						"error", decodeErr)
					msg.Ack()

					continue
				}

				if dispatchErr := b.dispatchLocal(b.subCtx, evt); dispatchErr != nil {
					// Handler error is logged and Acked. Nack would cause GoChannel
					// to retry the same message indefinitely (handler is deterministic),
					// deadlocking under BlockPublishUntilSubscriberAck. Consumers who
					// want retry semantics should wrap their handler with retry logic.
					b.logger.ErrorContext(b.subCtx, "watermill: dispatch failed",
						"event_type", evt.Type(), "error", dispatchErr)
					msg.Ack()

					continue
				}

				msg.Ack()
			case <-b.subCtx.Done():
				return
			}
		}
	}()
}
