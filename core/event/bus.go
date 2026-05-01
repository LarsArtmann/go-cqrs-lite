package event

import (
	"context"
	"io"
)

// Handler processes events.
type Handler func(ctx context.Context, event Event) error

// Bus defines the interface for event publishing and subscription.
// All implementations must support lifecycle management via io.Closer.
type Bus interface {
	io.Closer

	// Publish sends events to all registered handlers
	Publish(ctx context.Context, events ...Event) error

	// Subscribe registers a handler for specific event types
	Subscribe(eventType Type, handler Handler) error

	// SubscribeAll registers a handler for all event types
	SubscribeAll(handler Handler) error

	// Use adds middleware that wraps all event handlers
	Use(middleware ...Middleware) error
}

// Middleware wraps event handlers for cross-cutting concerns.
type Middleware func(Handler) Handler
