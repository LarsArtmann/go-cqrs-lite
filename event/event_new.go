package event

import (
	"encoding/json/jsontext"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// New creates a new event with a typed payload.
//
// If payload is []byte or json.RawMessage, it is used directly and the encoding
// defaults to [codec.EncodingJSON]. For all other types, the payload is marshaled
// using the codec provided via [WithCodec] (falling back to [DefaultCodec], which
// defaults to [codec.CBORCodec]), and the encoding is auto-stamped from the codec.
//
// Returns an error if payload is nil.
func New(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType id.AggregateType,
	version Version,
	payload any,
	opts ...Option,
) (*ImmutableEvent, error) {
	var c codec.Codec

	if len(opts) > 0 {
		probe := &ImmutableEvent{} //nolint:exhaustruct // probe: only opts field accessed

		for _, opt := range opts {
			opt(probe)
		}

		if probe.opts != nil && probe.opts.newCodec != nil {
			c = probe.opts.newCodec
		}
	}

	if c == nil {
		c = DefaultCodec
	}

	data, err := marshalPayload(payload, eventType, c)
	if err != nil {
		return nil, err
	}

	if err := validateEventParams(
		eventType,
		aggregateID,
		aggregateType,
		version,
		data,
	); err != nil {
		return nil, err
	}

	enc := c.Encoding()
	evt := buildEvent(eventType, aggregateID, aggregateType, version, data, opts)
	evt.encoding = enc

	return evt, nil
}

func marshalPayload(payload any, eventType Type, c codec.Codec) ([]byte, error) {
	if payload == nil {
		return nil, errorfamily.WrapRejection(
			ErrNilPayload,
			"event.nil_payload",
			"payload is required for event type "+string(eventType),
		)
	}

	switch v := payload.(type) {
	case []byte:
		return slices.Clone(v), nil
	case jsontext.Value:
		return slices.Clone(v), nil
	default:
		data, err := c.Encode(payload)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"event.marshal_payload_failed",
				"marshal payload for event type "+string(eventType),
			)
		}

		return data, nil
	}
}
