package event

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
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
// It accesses the field directly to avoid the defensive clone in Payload() —
// decoding only reads the bytes, never mutates.
func payloadForDecode(evt Event) []byte {
	return evt.payload
}

// PayloadReadOnly returns the event's payload bytes without cloning.
//
// The returned slice MUST NOT be mutated — it references the event's internal
// storage. Use this only for read-only operations (hashing, serialization,
// decoding). For any path that needs ownership of the bytes, use [Event.Payload]
// instead.
func PayloadReadOnly(evt Event) []byte {
	return payloadForDecode(evt)
}

// encodingForCopy returns the raw encoding field without normalization.
// Encoding() converts "" to "json", but copies should preserve the original
// field value to avoid altering the event's stored representation.
func encodingForCopy(evt Event) codec.Encoding {
	return evt.encoding
}

// DecodePayloads decodes multiple events' payloads into a slice of typed values.
// Validates encoding once for the batch instead of per-event.
// Returns an error at the first decode failure, indicating the index.
func DecodePayloads[T any](events []Event, c codec.Codec) ([]T, error) {
	result := make([]T, 0, len(events))

	for i, evt := range events {
		if err := validateEncodingMatch(evt, c); err != nil {
			return nil, WrapCorruption(
				err,
				"event.decode_payload_failed",
				"decode payload ["+strconv.Itoa(i)+"] for event "+string(evt.Type()),
			)
		}

		payload := payloadForDecode(evt)

		var target T
		if len(payload) > 0 {
			if err := c.Decode(payload, &target); err != nil {
				return nil, WrapCorruption(
					err,
					"event.decode_payload_failed",
					"decode payload ["+strconv.Itoa(i)+"] for event "+string(evt.Type()),
				)
			}
		}

		result = append(result, target)
	}

	return result, nil
}

func validateEncodingMatch(evt Event, c codec.Codec) error {
	evtEnc := evt.Encoding()
	codecEnc := c.Encoding()
	if evtEnc != codecEnc {
		return Newf(Rejection, "event.encoding_mismatch",
			"event encoding %q does not match codec encoding %q (decode payload for event %s)",
			evtEnc, codecEnc, evt.Type())
	}

	return nil
}
