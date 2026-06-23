package watermill

import (
	"io"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

// EventBusOption configures an EventBus.
type EventBusOption func(*EventBus)

// WithEventBusLogger sets the slog logger.
func WithEventBusLogger(logger *slog.Logger) EventBusOption {
	return func(b *EventBus) { b.logger = logger }
}

// WithEventBusTopic sets the Watermill topic (default: DefaultEventBusTopic).
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
