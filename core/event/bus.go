package event

import (
	"context"
	"io"
)

// Handler processes events.
type Handler func(ctx context.Context, event Event) error

// Publisher sends events without subscribing. Most consumers (repositories,
// outbox publishers) only need this interface — they never subscribe.
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
// All implementations must support lifecycle management via io.Closer.
//
// Bus composes Publisher and Subscriber so consumers can accept the smallest
// interface they need:
//
//	func ProcessEvents(sub event.Subscriber) { ... }  // only subscribes
//	func EmitEvents(pub event.Publisher) { ... }     // only publishes
type Bus interface {
	io.Closer
	Publisher
	Subscriber

	// Use adds middleware that wraps all event handlers
	Use(middleware ...Middleware) error

	// UsePublish adds middleware that wraps the Publish path.
	UsePublish(middleware ...PublishMiddleware) error
}

// Middleware wraps event handlers for cross-cutting concerns.
type Middleware func(Handler) Handler

// PublishMiddleware wraps the Publish method for cross-cutting concerns
// (logging, metrics, retry). Applied via Bus.UsePublish().
type PublishMiddleware func(Publisher) Publisher
