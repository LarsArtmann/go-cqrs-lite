package watermill

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// CommandBus is a full command.Bus implementation backed by a Watermill
// GoChannel. It replaces command.MemoryBus for single-process deployments
// that prefer Watermill's Ack/Nack lifecycle and buffer management. For
// multi-process deployments, inject a Kafka/NATS publisher+subscriber via
// WithCommandBackend.
type CommandBus struct {
	closed bool
	mu     sync.Mutex
	logger *slog.Logger
	topic  string

	publisher  message.Publisher
	subscriber message.Subscriber
	backend    io.Closer

	middleware    []command.Middleware
	cachedHandler command.Handler
	allHandlers   []command.Handler
	typeHandlers  map[command.Type][]command.Handler

	subCtx     context.Context //nolint:containedctx // lifecycle context; created from context.Background(), cancelled on Close
	subCancel  context.CancelFunc
	subStarted bool
}

var (
	_ command.Bus = (*CommandBus)(nil)
	_ io.Closer   = (*CommandBus)(nil)
)

// DefaultCommandBusTopic is the default Watermill topic used by CommandBus.
const DefaultCommandBusTopic = "cqrs.commands"

// NewCommandBus creates a Watermill-backed command.Bus. Without
// WithCommandBackend, uses a GoChannel for in-process pub/sub.
func NewCommandBus(opts ...CommandBusOption) *CommandBus {
	b := &CommandBus{
		logger:       slog.Default(),
		topic:        DefaultCommandBusTopic,
		typeHandlers: make(map[command.Type][]command.Handler),
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.publisher == nil || b.subscriber == nil {
		goChan := gochannel.NewGoChannel(
			gochannel.Config{
				Persistent:                     false,
				BlockPublishUntilSubscriberAck: true,
			},
			watermill.NopLogger{},
		)
		b.publisher = goChan
		b.subscriber = goChan
		b.backend = goChan
	}

	b.rebuildHandlerChain()

	return b
}

// Publish sends commands through to the Watermill topic.
func (b *CommandBus) Publish(_ context.Context, cmds ...command.Command) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return event.WrapInfrastructure(event.ErrBusClosed,
			"watermill.command_bus_publish", "command bus is closed")
	}

	topic := b.topic
	pub := b.publisher
	b.mu.Unlock()

	if len(cmds) == 0 {
		return nil
	}

	msgs := make([]*message.Message, 0, len(cmds))
	for _, cmd := range cmds {
		msgs = append(msgs, CommandToMessage(cmd))
	}

	if err := pub.Publish(topic, msgs...); err != nil {
		return event.WrapInfrastructure(err, "watermill.command_bus_publish",
			"publish to topic "+topic)
	}

	return nil
}

// Subscribe registers a handler for a specific command type.
func (b *CommandBus) Subscribe(cmdType command.Type, handler command.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.WrapInfrastructure(event.ErrBusClosed,
			"watermill.command_bus_subscribe", "command bus is closed")
	}

	b.typeHandlers[cmdType] = append(b.typeHandlers[cmdType], handler)
	b.rebuildHandlerChain()
	b.ensureSubscriptionLocked()

	return nil
}

// SubscribeAll registers a catch-all handler that receives every command.
func (b *CommandBus) SubscribeAll(handler command.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return event.WrapInfrastructure(event.ErrBusClosed,
			"watermill.command_bus_subscribe_all", "command bus is closed")
	}

	b.allHandlers = append(b.allHandlers, handler)
	b.rebuildHandlerChain()
	b.ensureSubscriptionLocked()

	return nil
}

// Use adds middleware that wraps all command handlers.
func (b *CommandBus) Use(mw ...command.Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, mw...)
	b.rebuildHandlerChain()

	return nil
}

// Close shuts down the backend. Safe to call multiple times.
func (b *CommandBus) Close() error {
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

// String returns a debug representation.
func (b *CommandBus) String() string {
	return fmt.Sprintf("watermill.CommandBus(topic=%s, closed=%v)", b.topic, b.closed)
}
