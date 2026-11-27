package watermill

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// SubscriberAdapter wraps a go-cqrs-lite event.Bus as a Watermill subscriber.
type SubscriberAdapter struct {
	bus event.Bus

	outputGate[*message.Message]
	outputCh chan *message.Message
	closeCh  chan struct{}
}

// NewSubscriberAdapter creates a Watermill subscriber backed by a go-cqrs-lite event.Bus.
func NewSubscriberAdapter(bus event.Bus) *SubscriberAdapter {
	return &SubscriberAdapter{
		bus:      bus,
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
		if err := a.offer(ctx, eventToMessage(evt), a.outputCh, a.closeCh); err != nil {
			return wrapOfferError(err, topic)
		}

		return nil
	}

	//cqrs-lint:ignore(C027) library code or intentional pattern
	if err := a.bus.Subscribe(event.Type(topic), handler); err != nil {
		//art-dupl:accept adapter twin subscribe-return shape — command/event adapters mirror by contract
		return nil, wrapSubscribeError(err, topic)
	}

	return a.outputCh, nil
}

// Close stops delivering events: buffered messages can still be read, then the
// output channel is closed.
//
// The event.Bus has no unsubscribe, so the handlers registered by Subscribe
// stay on the bus and reject every later event with [event.ErrBusClosed]
// (which the bus may log). If the bus implements Close, it is closed too.
func (a *SubscriberAdapter) Close() error {
	a.outputGate.close(a.closeCh, a.outputCh)

	if closer, ok := a.bus.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Subscriber = (*SubscriberAdapter)(nil)

// wrapOfferError classifies a refused offer: a closed adapter and a cancelled
// context both surface as Infrastructure errors carrying their cause.
func wrapOfferError(err error, topic string) error {
	if errors.Is(err, event.ErrBusClosed) {
		return errorfamily.WrapInfrastructure(event.ErrBusClosed,
			"watermill.topic_closed", fmt.Sprintf("topic %s", topic))
	}

	return errorfamily.WrapInfrastructure(err,
		"watermill.topic_cancelled", fmt.Sprintf("topic %s", topic))
}
