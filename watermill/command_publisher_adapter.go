package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
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
	return publishAll(
		context.Background(),
		func(ctx context.Context, cmd *command.BasicCommand) error { return a.publisher.Publish(ctx, cmd) },
		topic,
		messages,
		MessageToCommand,
		"watermill.publish_command_failed",
		func(cmd *command.BasicCommand) string { return "publish command " + string(cmd.Type()) },
	)
}

// Close closes the underlying publisher if it implements io.Closer.
func (a *CommandPublisherAdapter) Close() error {
	if closer, ok := a.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

var _ message.Publisher = (*CommandPublisherAdapter)(nil)
