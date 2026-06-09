package event

import (
	"fmt"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

// DecodePayload decodes an event's payload bytes into a typed value using
// the provided codec. This is the standard way to deserialize event data
// in event handlers and projectors.
//
// Returns a rejection error if the codec's encoding does not match the event's
// declared encoding (when both are non-empty and differ).
func DecodePayload[T any](evt Event, c codec.Codec) (T, error) {
	var zero T

	if err := validateEncodingMatch(evt, c); err != nil {
		return zero, err
	}

	payload := payloadForDecode(evt)
	if len(payload) == 0 {
		return zero, nil
	}

	var target T

	err := c.Decode(payload, &target)
	if err != nil {
		return zero, WrapCorruption(
			err,
			"event.decode_payload_failed",
			"decode payload for event "+string(evt.Type()),
		)
	}

	return target, nil
}

// payloadForDecode returns the raw event payload for read-only use in decoding.
// For *ImmutableEvent (the common case), it accesses the field directly to avoid
// the defensive clone in Payload() — decoding only reads the bytes, never mutates.
func payloadForDecode(evt Event) []byte {
	if ie, ok := evt.(*ImmutableEvent); ok {
		return ie.payload
	}

	return evt.Payload()
}

// DecodePayloads decodes multiple events' payloads into a slice of typed values.
// Returns an error at the first decode failure, indicating the index.
func DecodePayloads[T any](events []Event, c codec.Codec) ([]T, error) {
	result := make([]T, 0, len(events))

	for i, evt := range events {
		v, err := DecodePayload[T](evt, c)
		if err != nil {
			return nil, WrapCorruption(
				err,
				"event.decode_payload_failed",
				"decode payload ["+strconv.Itoa(i)+"] for event "+string(evt.Type()),
			)
		}

		result = append(result, v)
	}

	return result, nil
}

func validateEncodingMatch(evt Event, c codec.Codec) error {
	evtEnc := evt.Encoding()
	if evtEnc == "" || evtEnc == codec.EncodingJSON {
		return nil
	}

	codecEnc := c.Encoding()
	if codecEnc != evtEnc {
		return WrapRejection(
			fmt.Errorf("event encoding %q does not match codec encoding %q", evtEnc, codecEnc),
			"event.encoding_mismatch",
			"decode payload for event "+string(evt.Type()),
		)
	}

	return nil
}
