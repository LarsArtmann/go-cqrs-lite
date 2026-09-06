package watermill

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// SubscriberAdapter wraps a go-cqrs-lite event.Bus as a Watermill subscriber.
type SubscriberAdapter struct {
	bus        event.Bus
	handlers   map[string]event.Handler
	handlersMu sync.Mutex
	outputCh   chan *message.Message
	closeCh    chan struct{}
	closeMu    sync.Once
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
			return errorfamily.WrapInfrastructure(
				event.ErrBusClosed,
				"watermill.topic_closed",
				fmt.Sprintf("topic %s", topic),
			)
		case <-ctx.Done():
			return errorfamily.WrapInfrastructure(ctx.Err(),
				"watermill.topic_cancelled",
				fmt.Sprintf("topic %s", topic))
		}
	}

	//cqrs-lint:ignore(C027) library code or intentional pattern
	if err := a.bus.Subscribe(event.Type(topic), handler); err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"watermill.subscribe_failed",
			"subscribe to "+topic,
		)
	}

	registerSubscriberHandler(&a.handlersMu, a.handlers, topic, handler)

	return a.outputCh, nil
}

// Close closes the subscriber and unsubscribes all handlers.
func (a *SubscriberAdapter) Close() error {
	a.closeMu.Do(func() {
		close(a.closeCh)
		close(a.outputCh)
	})

	if closer, ok := a.bus.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Subscriber = (*SubscriberAdapter)(nil)
