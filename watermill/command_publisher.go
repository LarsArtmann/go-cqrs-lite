package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// CommandPublisher wraps a Watermill [message.Publisher] as a go-cqrs-lite
// [command.Publisher]. This is the cqrs → Watermill direction: commands
// produced by a command.Bus are published to a Watermill topic, where they
// can be routed to any Watermill-compatible destination.
//
// Usage:
//
//	pub := watermill.NewCommandPublisher(wmPublisher, "commands")
//	bus := command.NewMemoryBus()
//	bus.Publish(ctx, cmd) // → pub → Watermill topic
type CommandPublisher struct {
	publisher message.Publisher
	topic     string
}

// NewCommandPublisher creates a [command.Publisher] that publishes cqrs
// commands to the given Watermill topic.
func NewCommandPublisher(publisher message.Publisher, topic string) *CommandPublisher {
	return &CommandPublisher{publisher: publisher, topic: topic}
}

// Publish converts cqrs commands to Watermill messages and publishes them.
// Implements [command.Publisher].
func (p *CommandPublisher) Publish(_ context.Context, cmds ...command.Command) error {
	msgs := make([]*message.Message, 0, len(cmds))

	for _, cmd := range cmds {
		msgs = append(msgs, CommandToMessage(cmd))
	}

	if err := p.publisher.Publish(p.topic, msgs...); err != nil {
		return event.WrapInfrastructure(
			err, "watermill.publish_command_failed", "publish to topic "+p.topic,
		)
	}

	return nil
}

var _ command.Publisher = (*CommandPublisher)(nil)
