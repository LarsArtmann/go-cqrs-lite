package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

// CommandPublisherAdapter wraps a go-cqrs-lite command.Publisher as a
// Watermill publisher. This is the cqrs → Watermill direction: commands
// produced by a command.Bus are published to a Watermill topic.
type CommandPublisherAdapter struct {
	publisher command.Publisher
}

// NewCommandPublisherAdapter creates a Watermill publisher backed by a
// go-cqrs-lite command.Publisher.
func NewCommandPublisherAdapter(publisher command.Publisher) *CommandPublisherAdapter {
	return &CommandPublisherAdapter{publisher: publisher}
}

// Publish publishes Watermill messages as go-cqrs-lite commands.
// The topic is mapped to command type; all command fields are reconstructed
// from message metadata.
func (a *CommandPublisherAdapter) Publish(topic string, messages ...*message.Message) error {
	ctx := context.Background()

	for _, msg := range messages {
		cmd, err := MessageToCommand(topic, msg)
		if err != nil {
			return errorfamily.WrapCorruption(
				err,
				"watermill.convert_message_failed",
				"convert message "+msg.UUID,
			)
		}

		if err := a.publisher.Publish(ctx, cmd); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"watermill.publish_command_failed",
				"publish command "+string(cmd.Type()),
			)
		}
	}

	return nil
}

// Close closes the underlying publisher if it implements io.Closer.
func (a *CommandPublisherAdapter) Close() error {
	if closer, ok := a.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Publisher = (*CommandPublisherAdapter)(nil)
