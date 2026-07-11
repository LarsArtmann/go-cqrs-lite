package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
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
func (p *CommandPublisher) Publish(ctx context.Context, cmds ...command.Command) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "watermill.command.publish",
		cqrsotel.SpanKindProducer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrInt("cqrs.command.count", len(cmds)),
		),
	)
	defer span.End()

	if len(cmds) > 0 {
		attrs := cqrsotel.CommandAttrs(string(cmds[0].Type()), cmds[0].AggregateID())
		span.SetAttributes(attrs...)
	}

	msgs := make([]*message.Message, 0, len(cmds))

	for _, cmd := range cmds {
		msg := CommandToMessage(cmd)
		injectTraceContext(ctx, msg)
		msgs = append(msgs, msg)
	}

	if err := p.publisher.Publish(p.topic, msgs...); err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(
			err, "watermill.publish_command_failed", "publish to topic "+p.topic,
		)
	}

	return nil
}

var _ command.Publisher = (*CommandPublisher)(nil)
