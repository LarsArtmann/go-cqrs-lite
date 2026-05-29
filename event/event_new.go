package event

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/id"
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
	c := probeCodec(opts)

	data, err := marshalPayload(payload, eventType, c)
	if err != nil {
		return nil, err
	}

	allOpts := make([]Option, 0, len(opts)+1)
	allOpts = append(allOpts, WithEncoding(c.Encoding()))
	allOpts = append(allOpts, opts...)

	return NewEvent(eventType, aggregateID, aggregateType, version, data, allOpts...)
}

// probeCodec applies options to a zero-value ImmutableEvent to find a codec set
// via WithNewCodec. Returns JSONCodec if none found.
func probeCodec(opts []Option) codec.Codec {
	probe := &ImmutableEvent{}
	for _, opt := range opts {
		opt(probe)
	}

	if probe.newCodec != nil {
		return probe.newCodec
	}

	return codec.JSONCodec{}
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
		return v, nil
	case json.RawMessage:
		return v, nil
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
