package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// EventPublisher wraps a Watermill [message.Publisher] as a go-cqrs-lite
// [event.Publisher]. This is the cqrs → Watermill direction: events produced
// by the Decider are published to a Watermill topic, where they can be routed
// to any Watermill-compatible destination (GoChannel, Kafka, SQL, etc.).
//
// Combined with [SubscriberAdapter] (Watermill → cqrs) or [CatchUpSubscriber],
// this enables full bidirectional event flow through Watermill:
//
//	Decider → EventPublisher → Watermill topic → CatchUpSubscriber → Materialize
//
// Usage:
//
//	pub := watermill.NewEventPublisher(wmPublisher, "events")
//	repo, _ := decider.NewRepository(store, pub, decider)
type EventPublisher struct {
	publisher message.Publisher
	topic     string
}

// NewEventPublisher creates an [event.Publisher] that publishes cqrs events to
// the given Watermill topic via the given [message.Publisher].
//
// Each event is converted to a Watermill message using the same protocol as
// [PublisherAdapter] (reversible via [MessageToEvent]).
func NewEventPublisher(publisher message.Publisher, topic string) *EventPublisher {
	return &EventPublisher{publisher: publisher, topic: topic}
}

// Publish converts cqrs events to Watermill messages and publishes them.
// Implements [event.Publisher].
func (p *EventPublisher) Publish(ctx context.Context, events ...event.Event) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "watermill.event.publish",
		cqrsotel.SpanKindProducer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)),
		),
	)
	defer span.End()

	if len(events) > 0 {
		attrs := cqrsotel.EventAttrs(
			string(events[0].Type()),
			events[0].AggregateID(),
			string(events[0].AggregateType()),
		)
		span.SetAttributes(attrs...)
	}

	msgs := make([]*message.Message, 0, len(events))

	for _, evt := range events {
		msg := eventToMessage(evt)
		injectTraceContext(ctx, msg)
		msgs = append(msgs, msg)
	}

	if err := p.publisher.Publish(p.topic, msgs...); err != nil {
		cqrsotel.RecordError(span, err)

		return err //nolint:wrapcheck // transparent proxy
	}

	return nil
}

var _ event.Publisher = (*EventPublisher)(nil)
