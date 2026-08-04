package system

import (
	"context"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// simpleBus is a minimal in-process event bus. It implements event.Bus
// (Publisher + Subscriber + middleware). Handlers are invoked synchronously
// on the publishing goroutine.
//
// This is intentionally simple — no persistence, no retries, no async dispatch.
// For production message distribution, operators configure external bus drivers
// (NATS, Redis, Kafka) via the bus driver registry.
type simpleBus struct {
	mu            sync.RWMutex
	handlers      map[event.Type][]event.Handler
	allHandlers   []event.Handler
	middleware    []event.Middleware
	pubMiddleware []event.PublishMiddleware
}

func newSimpleBus() *simpleBus {
	return &simpleBus{
		handlers: make(map[event.Type][]event.Handler),
	}
}

// buildEventBus creates the event bus based on the deployment config.
// If the deployment configures a bus with a known driver, the driver factory
// is used. Otherwise a single simpleBus is created (D9).
func buildEventBus(deployment DeploymentConfig) event.Bus {
	// If a bus is explicitly configured, try the driver registry.
	for _, busCfg := range deployment.Buses {
		if busCfg.Driver == "" || busCfg.Driver == "gochannel" {
			continue
		}

		factory, err := lookupBusDriver(busCfg.Driver)
		if err != nil {
			continue // unknown driver, fall through to simpleBus
		}

		bus, err := factory(busCfg)
		if err != nil {
			continue // factory error, fall through to simpleBus
		}

		if eb, ok := bus.(event.Bus); ok {
			return eb
		}
	}

	return newSimpleBus()
}

// buildPublisher creates the publisher for the decider repository.
// If the source-of-truth has multiple Publish targets, returns a MultiBus
// wrapping a simpleBus for each target. Otherwise returns the local bus.
func buildPublisher(deployment DeploymentConfig, localBus event.Bus) event.Publisher {
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) && len(inst.Publish) > 1 {
			buses := make([]event.Publisher, len(inst.Publish))

			for i := range inst.Publish {
				buses[i] = newSimpleBus()
			}

			// Include the local bus so local subscribers still receive events.
			buses = append([]event.Publisher{localBus}, buses...)

			return NewMultiBus(buses...)
		}
	}

	return localBus
}

// Compile-time assertions.
var (
	_ event.Publisher = (*simpleBus)(nil)
	_ event.Bus       = (*simpleBus)(nil)
)

func (b *simpleBus) Publish(ctx context.Context, events ...event.Event) error {
	// Apply publish middleware chain.
	var publisher event.Publisher = event.PublisherFunc(func(ctx context.Context, evts ...event.Event) error {
		for _, evt := range evts {
			if err := b.dispatch(ctx, evt); err != nil {
				return err
			}
		}

		return nil
	})

	for _, v := range slices.Backward(b.pubMiddleware) {
		publisher = v(publisher)
	}

	return publisher.Publish(ctx, events...)
}

func (b *simpleBus) dispatch(ctx context.Context, evt event.Event) error {
	b.mu.RLock()

	handlers := make([]event.Handler, 0)

	// Typed handlers.
	if typed, ok := b.handlers[evt.Type()]; ok {
		handlers = append(handlers, typed...)
	}

	// Catch-all handlers.
	handlers = append(handlers, b.allHandlers...)

	// Clone middleware snapshot for independent handler calls.
	middleware := slices.Clone(b.middleware)

	b.mu.RUnlock()

	// Call each handler independently so one handler's error does not
	// prevent the others from executing. The first error is returned.
	var firstErr error

	for _, handler := range handlers {
		h := handler

		// Apply middleware chain for this handler.
		chain := h

		for _, v := range slices.Backward(middleware) {
			chain = v(chain)
		}

		if err := chain(ctx, evt); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (b *simpleBus) Subscribe(eventType event.Type, handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}

func (b *simpleBus) SubscribeAll(handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

func (b *simpleBus) Use(middleware ...event.Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)

	return nil
}

func (b *simpleBus) UsePublish(middleware ...event.PublishMiddleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pubMiddleware = append(b.pubMiddleware, middleware...)

	return nil
}
