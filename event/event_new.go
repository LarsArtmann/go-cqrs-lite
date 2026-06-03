package event

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// New creates a new event with a typed payload.
//
// If payload is []byte or json.RawMessage, it is used directly and the encoding
// defaults to [codec.EncodingJSON]. For all other types, the payload is marshaled
// using the codec provided via [WithNewCodec] (defaults to [codec.JSONCodec] if none
// is given), and the encoding is auto-stamped from the codec.
//
// Returns an error if payload is nil.
func New(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
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
		c = codec.JSONCodec{}
	}

	data, err := marshalPayload(payload, eventType, c)
	if err != nil {
		return nil, err
	}

	if err := validateEventParams(eventType, aggregateID, aggregateType, version, data); err != nil {
		return nil, err
	}

	enc := c.Encoding()
	evt := buildEvent(eventType, aggregateID, aggregateType, version, data, opts)
	evt.encoding = enc

	return evt, nil
}

func marshalPayload(payload any, eventType Type, c codec.Codec) ([]byte, error) {
	if payload == nil {
		return nil, WrapRejection(
			ErrNilPayload,
			"event.nil_payload",
			"payload is required for event type "+string(eventType),
		)
	}

	switch v := payload.(type) {
	case []byte:
		cloned := make([]byte, len(v))
		copy(cloned, v)
		return cloned, nil
	case json.RawMessage:
		cloned := make([]byte, len(v))
		copy(cloned, v)
		return cloned, nil
	default:
		data, err := c.Encode(payload)
		if err != nil {
			return nil, WrapCorruption(
				err,
				"event.marshal_payload_failed",
				"marshal payload for event type "+string(eventType),
			)
		}

		return data, nil
	}
}
