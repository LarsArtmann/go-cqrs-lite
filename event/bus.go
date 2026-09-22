package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
)

// Handler processes events.
type Handler func(ctx context.Context, event Event) error

// Publisher sends events without subscribing. Most consumers (repositories,
// projections) only need this interface — they never subscribe.
type Publisher interface {
	Publish(ctx context.Context, events ...Event) error
}

// PublisherFunc is a function adapter for Publisher.
type PublisherFunc func(ctx context.Context, events ...Event) error

// Publish calls the underlying function.
func (f PublisherFunc) Publish(ctx context.Context, events ...Event) error {
	return f(ctx, events...)
}

// Subscriber registers handlers for events. Projections and event processors
// only need this interface — they never publish.
type Subscriber interface {
	Subscribe(eventType Type, handler Handler) error
	SubscribeAll(handler Handler) error
}

// Bus defines the interface for event publishing and subscription.
// Implementations that own resources should implement io.Closer; callers
// type-assert when they need cleanup: `if c, ok := bus.(io.Closer); ok { c.Close() }`.
//
// Bus composes Publisher and Subscriber so consumers can accept the smallest
// interface they need:
//
//	func ProcessEvents(sub event.Subscriber) { ... }  // only subscribes
//	func EmitEvents(pub event.Publisher) { ... }     // only publishes
type Bus interface {
	Publisher
	Subscriber

	// Use adds middleware that wraps all event handlers
	Use(middleware ...Middleware) error

	// UsePublish adds middleware that wraps the Publish path.
	UsePublish(middleware ...PublishMiddleware) error
}

// Middleware wraps event handlers for cross-cutting concerns. It is an
// alias of the shared [dispatcher.Middleware] shape (E15 unification): one
// function value composes with every dispatcher and bus.
type Middleware = dispatcher.Middleware[Handler]

// PublishMiddleware wraps the Publish method for cross-cutting concerns
// (logging, metrics, retry). Applied via Bus.UsePublish(). Alias of the
// shared [dispatcher.Middleware] shape (E15 unification).
type PublishMiddleware = dispatcher.Middleware[Publisher]
