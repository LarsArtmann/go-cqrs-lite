package watermill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/core/event"
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
// The topic is mapped to event.Type.
func (a *PublisherAdapter) Publish(topic string, messages ...*message.Message) error {
	ctx := context.Background()

	for _, msg := range messages {
		evt, err := a.toEvent(topic, msg)
		if err != nil {
			return fmt.Errorf("convert message %s: %w", msg.UUID, err)
		}

		if err := a.publisher.Publish(ctx, evt); err != nil {
			return fmt.Errorf("publish event %s: %w", evt.Type(), err)
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

func (a *PublisherAdapter) toEvent(topic string, msg *message.Message) (event.Event, error) {
	// Decode event from Watermill message payload
	var evt event.ImmutableEvent

	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	return &evt, nil
}

var _ message.Publisher = (*PublisherAdapter)(nil)
