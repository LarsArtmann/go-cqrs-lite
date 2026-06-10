package event

import (
	"fmt"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type builder struct {
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	payload       []byte
	opts          []Option
}

func newBuilder(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
) *builder {
	return &builder{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		payload:       nil,
		opts:          nil,
	}
}

func (b *builder) WithPayload(payload []byte) *builder {
	b.payload = slices.Clone(payload)

	return b
}

func (b *builder) WithOptions(opts ...Option) *builder {
	b.opts = append(b.opts, opts...)

	return b
}

func (b *builder) WithCorrelationID(correlationID id.CorrelationID) *builder {
	b.opts = append(b.opts, WithCorrelationID(correlationID))

	return b
}

func (b *builder) WithCausationID(causationID id.CausationID) *builder {
	b.opts = append(b.opts, WithCausationID(causationID))

	return b
}

func (b *builder) WithUserID(userID id.UserID) *builder {
	b.opts = append(b.opts, WithUserID(userID))

	return b
}

func (b *builder) Build() (*ImmutableEvent, error) {
	err := validateEventParams(
		b.eventType,
		b.aggregateID,
		b.aggregateType,
		b.version,
		b.payload,
	)
	if err != nil {
		return nil, WrapCorruption(
			err,
			"event.build_failed",
			"build event "+string(b.eventType),
		)
	}

	return buildEvent(
		b.eventType,
		b.aggregateID,
		b.aggregateType,
		b.version,
		b.payload,
		b.opts,
	), nil
}

func (b *builder) MustBuild() *ImmutableEvent {
	evt, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("event.builder.MustBuild: %v", err))
	}

	return evt
}
