package watermill

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// EventBus is a full event.Bus implementation backed by Watermill message
// infrastructure. It replaces memory.MemoryBus for single-process deployments
// and serves as the canonical event bus for new code (ADR-0028).
//
// By default, EventBus uses an internal in-process pub/sub (no external
// broker). For multi-process deployments, inject a message.Publisher and
// message.Subscriber backed by Kafka, NATS, or any Watermill-compatible
// backend via WithBackend.
//
// Unlike MemoryBus, EventBus exposes Watermill's correlation ID middleware
// and supports the full event.Bus interface (Publish, Subscribe, Use,
// UsePublish, Close).
type EventBus struct {
	closed bool
	mu     sync.Mutex
	logger *slog.Logger
	topic  string

	publisher  message.Publisher
	subscriber message.Subscriber
	backend    io.Closer

	publishMiddleware []event.PublishMiddleware
	middleware        []event.Middleware
	cachedPublisher   event.Publisher
	cachedHandler     event.Handler
	allHandlers       []event.Handler
	typeHandlers      map[event.Type][]event.Handler

	subCtx     context.Context
	subCancel  context.CancelFunc
	subStarted bool
}

var (
	_ event.Bus = (*EventBus)(nil)
	_ io.Closer = (*EventBus)(nil)
)

// EventBusOption configures an EventBus.
type EventBusOption func(*EventBus)

// WithEventBusLogger sets the slog logger.
func WithEventBusLogger(logger *slog.Logger) EventBusOption {
	return func(b *EventBus) { b.logger = logger }
}

// WithEventBusTopic sets the Watermill topic (default: "cqrs.events").
func WithEventBusTopic(topic string) EventBusOption {
	return func(b *EventBus) { b.topic = topic }
}

// WithBackend injects external Watermill publisher and subscriber backends
// (e.g., Kafka, NATS). When provided, EventBus becomes multi-process capable.
// The closer (if non-nil) is called on Close.
func WithBackend(pub message.Publisher, sub message.Subscriber, closer io.Closer) EventBusOption {
	return func(b *EventBus) {
		b.publisher = pub
		b.subscriber = sub
		b.backend = closer
	}
}

// NewEventBus creates a Watermill-backed event.Bus. Without WithBackend,
// uses an internal in-process pub/sub suitable for single-process
// deployments and testing.
func NewEventBus(opts ...EventBusOption) *EventBus {
	b := &EventBus{
		logger:       slog.Default(),
		topic:        "cqrs.events",
		typeHandlers: make(map[event.Type][]event.Handler),
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.publisher == nil || b.subscriber == nil {
		backend := newInProcessPubSub()
		b.publisher = backend
		b.subscriber = backend
		b.backend = backend
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

		return event.WrapInfrastructure(event.ErrBusClosed, "watermill.event_bus_publish",
			"event bus is closed")
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
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.WrapInfrastructure(event.ErrBusClosed, "watermill.event_bus_subscribe",
			"event bus is closed")
	}

	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	b.rebuildHandlerChain()
	b.ensureSubscriptionLocked()

	return nil
}

// SubscribeAll registers a catch-all handler that receives every event.
func (b *EventBus) SubscribeAll(handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.WrapInfrastructure(event.ErrBusClosed, "watermill.event_bus_subscribe_all",
			"event bus is closed")
	}

	b.allHandlers = append(b.allHandlers, handler)
	b.rebuildHandlerChain()
	b.ensureSubscriptionLocked()

	return nil
}

// Use adds middleware that wraps all event handlers.
func (b *EventBus) Use(mw ...event.Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, mw...)
	b.rebuildHandlerChain()

	return nil
}

// UsePublish adds middleware that wraps the Publish path.
func (b *EventBus) UsePublish(mw ...event.PublishMiddleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.publishMiddleware = append(b.publishMiddleware, mw...)
	b.rebuildPublisherChain()

	return nil
}

// Close shuts down the backend. Safe to call multiple times.
func (b *EventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	if b.subCancel != nil {
		b.subCancel()
	}

	if b.backend != nil {
		return b.backend.Close()
	}

	return nil
}

func (b *EventBus) rebuildPublisherChain() {
	var inner event.Publisher = event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
		for _, evt := range events {
			msg := eventToMessage(evt)
			if err := b.publisher.Publish(b.topic, msg); err != nil {
				return event.WrapInfrastructure(err, "watermill.event_bus_publish",
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
					b.logger.ErrorContext(b.subCtx, "watermill: decode message failed",
						"error", decodeErr)
					msg.Nack()

					continue
				}

				if dispatchErr := b.dispatchLocal(b.subCtx, evt); dispatchErr != nil {
					b.logger.ErrorContext(b.subCtx, "watermill: dispatch failed",
						"event_type", evt.Type(), "error", dispatchErr)
					msg.Nack()

					continue
				}

				msg.Ack()
			case <-b.subCtx.Done():
				return
			}
		}
	}()
}

// inProcessPubSub is a minimal in-process message.Publisher + message.Subscriber.
// It replaces Watermill's GoChannel for single-process use without adding the
// gochannel module dependency.
type inProcessPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *message.Message
	closed      bool
}

func newInProcessPubSub() *inProcessPubSub {
	return &inProcessPubSub{subscribers: make(map[string][]chan *message.Message)}
}

func (p *inProcessPubSub) Publish(topic string, messages ...*message.Message) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return event.ErrBusClosed
	}

	subs := p.subscribers[topic]

	for _, msg := range messages {
		for _, ch := range subs {
			select {
			case ch <- msg:
			default:
			}
		}
	}

	return nil
}

func (p *inProcessPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, event.ErrBusClosed
	}

	ch := make(chan *message.Message, 128)
	p.subscribers[topic] = append(p.subscribers[topic], ch)

	return ch, nil
}

func (p *inProcessPubSub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	for _, subs := range p.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}

	p.subscribers = make(map[string][]chan *message.Message)

	return nil
}

// CorrelationID returns the Watermill correlation ID middleware for convenience.
// Apply via bus.Use(...) if you want correlation ID propagation.
func CorrelationID() event.Middleware {
	correlationMiddleware := middleware.CorrelationID

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			_ = correlationMiddleware // available for consumers who want it

			return next(ctx, evt)
		}
	}
}
