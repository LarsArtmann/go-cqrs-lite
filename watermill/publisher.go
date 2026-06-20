package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// PublisherAdapter wraps a go-cqrs-lite event.Publisher as a Watermill publisher.
type PublisherAdapter struct {
	publisher event.Publisher
}

// NewPublisherAdapter creates a Watermill publisher backed by a go-cqrs-lite event.Publisher.
func NewPublisherAdapter(publisher event.Publisher) *PublisherAdapter {
	return &PublisherAdapter{publisher: publisher}
}

// Publish publishes Watermill messages as go-cqrs-lite events.
// The topic is mapped to event.Type; all event fields are reconstructed from message metadata.
func (a *PublisherAdapter) Publish(topic string, messages ...*message.Message) error {
	ctx := context.Background()

	for _, msg := range messages {
		evt, err := MessageToEvent(topic, msg)
		if err != nil {
			return event.WrapCorruption(
				err,
				"watermill.convert_message_failed",
				"convert message "+msg.UUID,
			)
		}

		if err := a.publisher.Publish(ctx, evt); err != nil {
			return event.WrapInfrastructure(
				err,
				"watermill.publish_event_failed",
				"publish event "+string(evt.Type()),
			)
		}
	}

	return nil
}

// Close closes the underlying publisher.
func (a *PublisherAdapter) Close() error {
	if closer, ok := a.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Publisher = (*PublisherAdapter)(nil)
