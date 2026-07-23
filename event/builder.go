package event

import (
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type builder struct {
	eventType     Type
	streamID   id.StreamID
	streamType id.StreamType
	version       Version
	payload       []byte
	opts          []Option
}

func newBuilder(
	eventType Type,
	streamID id.StreamID,
	streamType id.StreamType,
	version Version,
) *builder {
	return &builder{
		eventType:     eventType,
		streamID:   streamID,
		streamType: streamType,
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
		b.streamID,
		b.streamType,
		b.version,
		b.payload,
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"event.build_failed",
			"build event "+string(b.eventType),
		)
	}

	return buildEvent(
		b.eventType,
		b.streamID,
		b.streamType,
		b.version,
		b.payload,
		b.opts,
	), nil
}
