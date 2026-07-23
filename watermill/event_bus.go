package watermill

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// EventBus is a full event.Bus implementation backed by a Watermill GoChannel.
// It replaces memory.MemoryBus for single-process deployments and serves as
// the canonical event bus for new code (ADR-0028).
//
// GoChannel provides persistent in-process pub/sub with proper message routing,
// Ack/Nack lifecycle, and buffer management. For multi-process deployments,
// inject a Kafka/NATS publisher+subscriber via WithBackend.
type EventBus struct {
	subscriptionState

	closed bool
	mu     sync.Mutex

	logger *slog.Logger
	topic  string

	backend    io.Closer
	publisher  message.Publisher
	subscriber message.Subscriber

	publishMiddleware []event.PublishMiddleware
	middleware        []event.Middleware
	cachedPublisher   event.Publisher
	cachedHandler     event.Handler
	allHandlers       []event.Handler
	typeHandlers      map[event.Type][]event.Handler
}

var (
	_ event.Bus = (*EventBus)(nil)
	_ io.Closer = (*EventBus)(nil)
)

// MessageSubscriber exposes the internal Watermill [message.Subscriber]
// (GoChannel by default). This is needed for CatchUpSubscriber, which
// requires a message.Subscriber for the live-delivery phase.
func (b *EventBus) MessageSubscriber() message.Subscriber { return b.subscriber }

// DefaultEventBusTopic is the default Watermill topic used by EventBus.
const DefaultEventBusTopic = "cqrs.events"

// NewEventBus creates a Watermill-backed event.Bus. Without WithBackend,
// uses a GoChannel for persistent in-process pub/sub suitable for
// single-process deployments and testing.
func NewEventBus(opts ...EventBusOption) *EventBus {
	b := &EventBus{
		logger:       slog.Default(),
		topic:        DefaultEventBusTopic,
		typeHandlers: make(map[event.Type][]event.Handler),
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.publisher == nil || b.subscriber == nil {
		goChan := gochannel.NewGoChannel(
			gochannel.Config{
				// Persistent is intentionally false: CatchUpSubscriber handles
				// historical replay from the journal (ordered). Persistent mode
				// delivers buffered messages via separate goroutines per message
				// (unordered) and registers the subscriber only after replay
				// completes, creating a delivery gap.
				//
				// BlockPublishUntilSubscriberAck ensures ordered live delivery:
				// Publish blocks until the subscriber acks, so consecutive
				// publishes cannot race on the output channel.
				Persistent:                     false,
				BlockPublishUntilSubscriberAck: true,
			},
			watermill.NopLogger{},
		)
		b.publisher = goChan
		b.subscriber = goChan
		b.backend = goChan
	}

	b.rebuildPublisherChain()
	b.rebuildHandlerChain()

	return b
}

// Publish sends events through the middleware chain to the Watermill topic.
func (b *EventBus) Publish(ctx context.Context, events ...event.Event) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return errBusClosed("watermill.event_bus_publish", "event")
	}

	pub := b.cachedPublisher
	b.mu.Unlock()

	if len(events) == 0 {
		return nil
	}

	return pub.Publish(ctx, events...)
}

// Subscribe registers a handler for a specific event type.
func (b *EventBus) Subscribe(eventType event.Type, handler event.Handler) error {
	return registerTypedHandler(&b.mu, b.closed, b.typeHandlers, eventType, handler,
		"watermill.event_bus_subscribe", "event",
		b.rebuildHandlerChain, b.ensureSubscriptionLocked)
}

// SubscribeAll registers a catch-all handler that receives every event.
func (b *EventBus) SubscribeAll(handler event.Handler) error {
	return registerAllHandler(&b.mu, b.closed, &b.allHandlers, handler,
		"watermill.event_bus_subscribe_all", "event",
		b.rebuildHandlerChain, b.ensureSubscriptionLocked)
}

// Use adds middleware that wraps all event handlers.
func (b *EventBus) Use(mw ...event.Middleware) error {
	withLockedModify(
		&b.mu,
		func() { b.middleware = append(b.middleware, mw...) },
		b.rebuildHandlerChain,
	)

	return nil
}

// UsePublish adds middleware that wraps the Publish path.
func (b *EventBus) UsePublish(mw ...event.PublishMiddleware) error {
	withLockedModify(
		&b.mu,
		func() { b.publishMiddleware = append(b.publishMiddleware, mw...) },
		b.rebuildPublisherChain,
	)

	return nil
}

// Close shuts down the backend. Safe to call multiple times.
func (b *EventBus) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()

		return nil
	}

	b.closed = true

	b.shutdown()

	backend := b.backend

	b.mu.Unlock()

	// Close the backend outside the lock to avoid blocking the dispatch
	// goroutine, which may be finishing an in-flight message via dispatchLocal.
	if backend != nil {
		return backend.Close()
	}

	return nil
}
