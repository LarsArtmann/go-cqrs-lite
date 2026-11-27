package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// CommandSubscriberAdapter wraps a go-cqrs-lite command.Subscriber as a
// Watermill subscriber. Commands received from the subscriber are delivered
// as Watermill messages on a single shared output channel.
type CommandSubscriberAdapter struct {
	subscriber command.Subscriber

	outputGate[*message.Message]
	outputCh chan *message.Message
	closeCh  chan struct{}
}

// NewCommandSubscriberAdapter creates a Watermill subscriber backed by a
// go-cqrs-lite command.Subscriber.
func NewCommandSubscriberAdapter(subscriber command.Subscriber) *CommandSubscriberAdapter {
	return &CommandSubscriberAdapter{
		subscriber: subscriber,
		outputCh:   make(chan *message.Message, 100),
		closeCh:    make(chan struct{}),
	}
}

// Subscribe creates a subscription for the given topic (mapped to command type).
func (a *CommandSubscriberAdapter) Subscribe(
	_ context.Context,
	topic string,
) (<-chan *message.Message, error) {
	handler := func(ctx context.Context, cmd command.Command) error {
		if err := a.offer(ctx, CommandToMessage(cmd), a.outputCh, a.closeCh); err != nil {
			return wrapOfferError(err, topic)
		}

		return nil
	}

	//cqrs-lint:ignore(C027) library code or intentional pattern
	if err := a.subscriber.Subscribe(command.Type(topic), handler); err != nil {
		return nil, wrapSubscribeError(err, topic)
	}

	return a.outputCh, nil
}

// Close stops delivering commands: buffered messages can still be read, then
// the output channel is closed.
//
// Handlers registered by Subscribe stay on the command.Subscriber (it has no
// unsubscribe) and reject every later command with [event.ErrBusClosed].
// If the subscriber implements Close, it is closed too.
func (a *CommandSubscriberAdapter) Close() error {
	a.outputGate.close(a.closeCh, a.outputCh)

	if closer, ok := a.subscriber.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Subscriber = (*CommandSubscriberAdapter)(nil)
