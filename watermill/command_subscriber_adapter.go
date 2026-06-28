package watermill

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// CommandSubscriberAdapter wraps a go-cqrs-lite command.Subscriber as a
// Watermill subscriber. Commands received from the subscriber are delivered
// as Watermill messages on a single shared output channel.
type CommandSubscriberAdapter struct {
	subscriber command.Subscriber
	handlers   map[string]command.Handler
	handlersMu sync.Mutex
	outputCh   chan *message.Message
	closeCh    chan struct{}
	closeOnce  sync.Once
}

// NewCommandSubscriberAdapter creates a Watermill subscriber backed by a
// go-cqrs-lite command.Subscriber.
func NewCommandSubscriberAdapter(subscriber command.Subscriber) *CommandSubscriberAdapter {
	return &CommandSubscriberAdapter{
		subscriber: subscriber,
		handlers:   make(map[string]command.Handler),
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
		msg := CommandToMessage(cmd)

		select {
		case a.outputCh <- msg:
			return nil
		case <-a.closeCh:
			return event.WrapInfrastructure(
				event.ErrBusClosed,
				"watermill.topic_closed",
				fmt.Sprintf("topic %s", topic),
			)
		case <-ctx.Done():
			return event.WrapInfrastructure(
				ctx.Err(),
				"watermill.topic_cancelled",
				fmt.Sprintf("topic %s", topic),
			)
		}
	}

	if err := a.subscriber.Subscribe(command.Type(topic), handler); err != nil {
		return nil, event.WrapInfrastructure(
			err, "watermill.subscribe_failed", "subscribe to "+topic,
		)
	}

	a.handlersMu.Lock()
	a.handlers[topic] = handler
	a.handlersMu.Unlock()

	return a.outputCh, nil
}

// Close closes the subscriber and the output channel.
func (a *CommandSubscriberAdapter) Close() error {
	a.closeOnce.Do(func() {
		close(a.closeCh)
		close(a.outputCh)
	})

	if closer, ok := a.subscriber.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Subscriber = (*CommandSubscriberAdapter)(nil)
