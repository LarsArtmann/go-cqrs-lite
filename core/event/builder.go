package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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
	b.payload = payload

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
	evt, err := NewEvent(
		b.eventType,
		b.aggregateID,
		b.aggregateType,
		b.version,
		b.payload,
		b.opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("build event %s: %w", b.eventType, err)
	}

	return evt, nil
}

func (b *builder) MustBuild() *ImmutableEvent {
	evt, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("event.builder.MustBuild: %v", err))
	}

	return evt
}
