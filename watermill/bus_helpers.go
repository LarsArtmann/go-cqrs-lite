package watermill

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// subscriptionState holds the lifecycle fields for the background subscriber
// goroutine. Both CommandBus and EventBus embed this struct so they share the
// same idempotent start/shutdown logic without duplicating the guard, subscribe,
// and teardown boilerplate.
type subscriptionState struct {
	subCtx     context.Context //nolint:containedctx // lifecycle context for the background subscriber goroutine; created from context.Background(), cancelled on shutdown
	subCancel  context.CancelFunc
	subStarted bool
}

// ensureStarted starts the background subscriber goroutine if it hasn't been
// started yet. The run callback receives the lifecycle context and the message
// channel, and contains the type-specific decode/dispatch loop. Idempotent:
// subsequent calls are no-ops.
func (s *subscriptionState) ensureStarted(
	subscriber message.Subscriber,
	topic string,
	logger *slog.Logger,
	run func(ctx context.Context, msgs <-chan *message.Message),
) {
	if s.subStarted {
		return
	}

	s.subCtx, s.subCancel = context.WithCancel(context.Background())

	msgs, err := subscriber.Subscribe(s.subCtx, topic)
	if err != nil {
		logger.ErrorContext(s.subCtx, "watermill: subscribe failed",
			"error", err, "topic", topic)
		s.subCancel()
		s.subCtx = nil
		s.subCancel = nil

		return
	}

	s.subStarted = true

	go run(s.subCtx, msgs)
}

// shutdown cancels the lifecycle context, stopping the background goroutine.
func (s *subscriptionState) shutdown() {
	if s.subCancel != nil {
		s.subCancel()
	}
}

// errBusClosed wraps event.ErrBusClosed for the given Watermill component and
// domain noun. Shared by CommandBus and EventBus methods so the error
// wrapping pattern stays consistent.
func errBusClosed(component, noun string) error {
	return errorfamily.WrapInfrastructure(event.ErrBusClosed,
		component, noun+" bus is closed")
}

// registerTypedHandler is the shared subscribe logic for both CommandBus and
// EventBus. It locks the bus, checks the closed guard, appends the handler to
// the type-keyed map, rebuilds the handler chain, and ensures the background
// subscription is running.
func registerTypedHandler[K ~string, H any](
	mu *sync.Mutex,
	closed bool,
	handlers map[K][]H,
	key K,
	handler H,
	component, noun string,
	rebuild func(),
	ensure func(),
) error {
	mu.Lock()
	defer mu.Unlock()

	if closed {
		return errBusClosed(component, noun)
	}

	handlers[key] = append(handlers[key], handler)
	rebuild()
	ensure()

	return nil
}

// registerAllHandler is the shared SubscribeAll logic for both CommandBus and
// EventBus. It takes a pointer to the allHandlers slice so the append updates
// the struct field.
func registerAllHandler[H any](
	mu *sync.Mutex,
	closed bool,
	handlers *[]H,
	handler H,
	component, noun string,
	rebuild func(),
	ensure func(),
) error {
	mu.Lock()
	defer mu.Unlock()

	if closed {
		return errBusClosed(component, noun)
	}

	*handlers = append(*handlers, handler)
	rebuild()
	ensure()

	return nil
}

// appendMiddleware is the shared Use/UsePublish logic for both CommandBus and
// EventBus. It locks the bus, appends the middleware, and rebuilds the chain.
// Unlike registerAllHandler, it does NOT check the closed guard or call ensure
// because adding middleware after close is harmless (Close tears down the
// subscription goroutine, not the middleware chain) and middleware does not
// require a running subscription.
func appendMiddleware[M any](
	mu *sync.Mutex,
	middleware *[]M,
	mw []M,
	rebuild func(),
) {
	mu.Lock()
	defer mu.Unlock()

	*middleware = append(*middleware, mw...)
	rebuild()
}

// registerSubscriberHandler stores a topic→handler mapping under a mutex.
// Shared between SubscriberAdapter and CommandSubscriberAdapter.
func registerSubscriberHandler[H any](
	mu *sync.Mutex,
	handlers map[string]H,
	topic string,
	handler H,
) {
	mu.Lock()
	handlers[topic] = handler
	mu.Unlock()
}

// dispatchCached reads a handler function under a mutex and invokes it outside
// the lock. Shared between CommandBus.dispatchLocal and EventBus.dispatchLocal.
func dispatchCached[M any](
	mu *sync.Mutex,
	cached func(context.Context, M) error,
	ctx context.Context,
	msg M,
) error {
	mu.Lock()
	handler := cached
	mu.Unlock()

	return handler(ctx, msg)
}
