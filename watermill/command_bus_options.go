package watermill

import (
	"io"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

// CommandBusOption configures a CommandBus.
type CommandBusOption func(*CommandBus)

// WithCommandBusLogger sets the slog logger.
func WithCommandBusLogger(logger *slog.Logger) CommandBusOption {
	return func(b *CommandBus) { b.logger = logger }
}

// WithCommandBusTopic sets the Watermill topic (default: DefaultCommandBusTopic).
func WithCommandBusTopic(topic string) CommandBusOption {
	return func(b *CommandBus) { b.topic = topic }
}

// WithCommandBackend injects external Watermill publisher and subscriber
// backends (e.g., Kafka, NATS). When provided, CommandBus becomes
// multi-process capable. The closer (if non-nil) is called on Close.
func WithCommandBackend(
	pub message.Publisher,
	sub message.Subscriber,
	closer io.Closer,
) CommandBusOption {
	return func(b *CommandBus) {
		b.publisher = pub
		b.subscriber = sub
		b.backend = closer
	}
}
