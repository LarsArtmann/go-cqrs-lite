package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
	return publishAll(
		context.Background(),
		func(ctx context.Context, evt event.Event) error { return a.publisher.Publish(ctx, evt) },
		topic,
		messages,
		MessageToEvent,
		wrapEventPublishFailure,
	)
}

// wrapEventPublishFailure owns the adapter's Infrastructure code at a
// literal call site — the taxonomy gate extracts codes from errorfamily
// calls, not helper parameters.
func wrapEventPublishFailure(err error, evt event.Event) error {
	return errorfamily.WrapInfrastructure(
		err,
		"watermill.publish_event_failed",
		"publish event "+string(evt.Type()),
	)
}

// Close closes the underlying publisher.
func (a *PublisherAdapter) Close() error {
	if closer, ok := a.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Publisher = (*PublisherAdapter)(nil)
