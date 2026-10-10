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
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// CommandBus is a full command.Bus implementation backed by a Watermill
// GoChannel. It replaces command.MemoryBus for single-process deployments
// that prefer Watermill's Ack/Nack lifecycle and buffer management. For
// multi-process deployments, inject a Kafka/NATS publisher+subscriber via
// WithCommandBackend.
type CommandBus struct {
	subscriptionState

	closed bool
	mu     sync.Mutex

	// publishWG drains in-flight Publish calls in Close: a publish that
	// passed the closed guard must reach the backend BEFORE Close closes
	// it. Without the drain, a publish straddling the backend close fails
	// with the transport's RAW closed error (watermill's "Pub/Sub closed"),
	// which errors.Is(event.ErrBusClosed) cannot recognize — the typed
	// bus-closed contract consumers match on.
	publishWG sync.WaitGroup

	logger *slog.Logger
	topic  string

	publisher  message.Publisher
	subscriber message.Subscriber
	backend    io.Closer

	middleware    []command.Middleware
	cachedHandler command.Handler
	allHandlers   []command.Handler
	typeHandlers  map[command.Type][]command.Handler
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
//
// A context marked as inside a synchronous bus delivery (see
// event.ContextInDelivery) is rejected with [ErrReentrantPublish] instead
// of deadlocking: the nested synchronous publish would block forever on the
// per-topic subscriber lock the delivering publish still holds
// (BlockPublishUntilSubscriberAck). Escape asynchronously and clear the
// mark (event.WithoutDeliveryMark, deriver.WithAsyncDispatch).
func (b *CommandBus) Publish(ctx context.Context, cmds ...command.Command) error {
	if event.ContextInDelivery(ctx) {
		return fmt.Errorf(
			"%w: nested command publish from a synchronous delivery handler",
			ErrReentrantPublish,
		)
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return errBusClosed("watermill.command_bus_publish", "command")
	}

	topic := b.topic
	pub := b.publisher
	//art-dupl:accept command/event bus publish-tail twins (event_bus.go): independent payload types keep the buses decoupled, the WaitGroup handshake is deliberately parallel
	b.publishWG.Add(1)
	b.mu.Unlock()

	defer b.publishWG.Done()

	if len(cmds) == 0 {
		return nil
	}

	msgs := make([]*message.Message, 0, len(cmds))
	for _, cmd := range cmds {
		msgs = append(msgs, CommandToMessage(cmd))
	}

	if err := pub.Publish(topic, msgs...); err != nil {
		return errorfamily.WrapInfrastructure(err, "watermill.command_bus_publish",
			"publish to topic "+topic)
	}

	return nil
}

// Subscribe registers a handler for a specific command type.
func (b *CommandBus) Subscribe(cmdType command.Type, handler command.Handler) error {
	return registerTypedHandler(&b.mu, b.closed, b.typeHandlers, cmdType, handler,
		"watermill.command_bus_subscribe", "command",
		b.rebuildHandlerChain, b.ensureSubscriptionLocked)
}

// SubscribeAll registers a catch-all handler that receives every command.
func (b *CommandBus) SubscribeAll(handler command.Handler) error {
	return registerAllHandler(&b.mu, b.closed, &b.allHandlers, handler,
		"watermill.command_bus_subscribe_all", "command",
		b.rebuildHandlerChain, b.ensureSubscriptionLocked)
}

// Use adds middleware that wraps all command handlers.
func (b *CommandBus) Use(mw ...command.Middleware) error {
	withLockedModify(
		&b.mu,
		func() { b.middleware = append(b.middleware, mw...) },
		b.rebuildHandlerChain,
	)

	return nil
}

// Close shuts down the backend. Safe to call multiple times.
func (b *CommandBus) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()

		//art-dupl:accept close-once latch idiom — bus Close contract, deliberately not merged (lock discipline differs)
		return nil
	}

	b.closed = true

	b.shutdown()

	backend := b.backend

	b.mu.Unlock()

	// Drain in-flight publishes BEFORE closing the backend, so a publish
	// that passed the closed guard never meets a closed transport (that
	// leaks the transport's raw closed error, unrecognizable as the typed
	// bus-closed contract). Publishes stay concurrent — the drain only
	// delays the backend close, not other publishes.
	b.publishWG.Wait()

	if backend != nil {
		return backend.Close()
	}

	return nil
}
