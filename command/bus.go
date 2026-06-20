package command

import (
	"context"
)

// Publisher publishes commands to subscribers.
// This is the command-side equivalent of event.Publisher.
type Publisher interface {
	Publish(ctx context.Context, cmds ...Command) error
}

// Subscriber subscribes to commands by type or to all commands.
// This is the command-side equivalent of event.Subscriber.
type Subscriber interface {
	Subscribe(cmdType Type, handler Handler) error
	SubscribeAll(handler Handler) error
}

// Bus is the full command pub/sub interface with middleware support.
// It mirrors event.Bus for command-side reactive dispatch.
//
// Use cases:
//   - Decouple command submission from handling (queue-based dispatch)
//   - Inter-service command routing (microservice orchestration)
//   - Command audit logging (subscribe to all commands)
//   - Saga coordination (commands triggered by events)
type Bus interface {
	Publisher
	Subscriber
	Use(middleware ...Middleware) error
}

// PublishMiddleware wraps the publish path (analogous to event.PublishMiddleware).
// Use it for concerns that should apply to outgoing commands:
// signing, encryption, tracing, metrics.
type PublishMiddleware func(Publisher) Publisher

// PublisherFunc is a function adapter for Publisher.
type PublisherFunc func(ctx context.Context, cmds ...Command) error

func (f PublisherFunc) Publish(
	ctx context.Context,
	cmds ...Command,
) error {
	return f(ctx, cmds...)
}
