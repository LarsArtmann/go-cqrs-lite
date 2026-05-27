package watermill

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SubscriberAdapter wraps a go-cqrs-lite event.Bus as a Watermill subscriber.
type SubscriberAdapter struct {
	bus      event.Bus
	handlers map[string]event.Handler
	outputCh chan *message.Message
	closeCh  chan struct{}
}

// NewSubscriberAdapter creates a Watermill subscriber backed by a go-cqrs-lite event.Bus.
func NewSubscriberAdapter(bus event.Bus) *SubscriberAdapter {
	return &SubscriberAdapter{
		bus:      bus,
		handlers: make(map[string]event.Handler),
		outputCh: make(chan *message.Message, 100),
		closeCh:  make(chan struct{}),
	}
}

// Subscribe creates a subscription for the given topic (mapped to event.Type).
func (a *SubscriberAdapter) Subscribe(
	_ context.Context,
	topic string,
) (<-chan *message.Message, error) {
	handler := func(ctx context.Context, evt event.Event) error {
		msg := eventToMessage(evt)

		select {
		case a.outputCh <- msg:
			return nil
		case <-a.closeCh:
			return fmt.Errorf("subscriber closed")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := a.bus.Subscribe(event.Type(topic), handler); err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", topic, err)
	}

	a.handlers[topic] = handler

	return a.outputCh, nil
}

// Close closes the subscriber and unsubscribes all handlers.
func (a *SubscriberAdapter) Close() error {
	close(a.closeCh)
	close(a.outputCh)

	if closer, ok := a.bus.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Subscriber = (*SubscriberAdapter)(nil)
