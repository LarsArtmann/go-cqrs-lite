package event

import (
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

// DefaultCodec is the codec used by [New] when no [WithCodec] option is
// provided. It defaults to [codec.JSONCodec] for backwards compatibility.
//
// To adopt CBOR for all events created via [New] without passing
// [WithCodec] on every call, set this once at program startup:
//
//	event.DefaultCodec = codec.CBORCodec{}
//
// Concurrency contract: set this ONCE at program startup, BEFORE any
// goroutine creates events. Like [net/http.DefaultClient], it is a plain
// package-level variable — concurrent reads after startup are safe, but
// concurrent read+write is a data race. Do not mutate it after events are
// in flight.
//
// This is a package-level default (like [net/http.DefaultClient]): it affects
// every [New] call in the process that does not override it with [WithCodec].
// Events created with an explicit [WithCodec] are unaffected. Because the
// encoding is stamped on each event ([ImmutableEvent.Encoding]), mixed streams
// of JSON and CBOR events decode correctly via [DecodePayloadAuto], which
// dispatches based on the per-event encoding stamp.
//
// Changing this after events have been created with the old default does NOT
// alter existing events; it only affects subsequently created ones.
var DefaultCodec codec.Codec = codec.JSONCodec{}

// DecodePayloadAuto decodes an event's payload into a typed value using the
// codec that matches the event's declared encoding ([ImmutableEvent.Encoding]).
// It resolves the codec via [codec.ForEncoding], so JSON events use
// [codec.JSONCodec] and CBOR events use [codec.CBORCodec] — automatically.
//
// This is the correct function for mixed-stream decoding: when an event store
// contains events created under different codec defaults (e.g. a JSON→CBOR
// migration), DecodePayloadAuto picks the right codec per event. Unlike
// [DecodePayload], it does NOT require the caller to know or pass the codec.
//
// Returns an error if the event's encoding has no built-in codec (e.g. "raw",
// "encrypted") — in that case, the caller must handle decoding manually.
func DecodePayloadAuto[T any](evt Event) (T, error) {
	var zero T

	c, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return zero, errorfamily.WrapCorruption(
			err,
			"event.decode_payload_auto_no_codec",
			"no built-in codec for encoding "+string(evt.Encoding())+
				" (decode payload for event "+string(evt.Type())+")",
		)
	}

	return DecodePayload[T](evt, c)
}

// DecodePayload decodes an event's payload bytes into a typed value using
// the provided codec. This is the standard way to deserialize event data
// in event handlers and projectors.
//
// Returns a rejection error if the codec's encoding does not match the event's
// declared encoding. For mixed JSON+CBOR streams, prefer [DecodePayloadAuto],
// which dispatches based on the event's encoding stamp.
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
		return zero, errorfamily.WrapCorruption(
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
// Each event's encoding is validated against the codec before decoding.
// Returns an error at the first decode failure, indicating the index.
func DecodePayloads[T any](events []Event, c codec.Codec) ([]T, error) {
	result := make([]T, 0, len(events))

	for i, evt := range events {
		if err := validateEncodingMatch(evt, c); err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"event.decode_payload_failed",
				"decode payload ["+strconv.Itoa(i)+"] for event "+string(evt.Type()),
			)
		}

		payload := payloadForDecode(evt)

		var target T
		if len(payload) > 0 {
			if err := c.Decode(payload, &target); err != nil {
				return nil, errorfamily.WrapCorruption(
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
		return errorfamily.Newf(Rejection, "event.encoding_mismatch",
			"event encoding %q does not match codec encoding %q (decode payload for event %s)",
			evtEnc, codecEnc, evt.Type())
	}

	return nil
}
