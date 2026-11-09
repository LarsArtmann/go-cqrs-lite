package metaengine

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ApplyEncoded processes an encoded event payload through all queries.
// The eventType identifies which fold to invoke. Payload may be JSON or CBOR:
// the codec is detected from the bytes, and each fold decodes the bytes with
// the matching codec into its own sample type. The payload of an event built
// with [event.New] (CBOR by default) therefore applies without a manual decode.
//
// The apply is a full pipeline citizen — identical to Store.Apply except the
// payload arrives undecoded: it is metered, hook-observed, recorded to the
// attached EventLog, replicated to shadow engines, and counted by the
// synthetic-record advisory (Store.Apply's Doctor caveat applies here too —
// only a Type-only Record can be synthesized, so OnRecord folds see empty
// StreamID/Version; use ApplyEncodedRecord to carry the full Record).
//
// Encodings outside the JSON/CBOR pair require a manual decode followed by
// Store.Apply.
//
// Example with event.Event:
//
//	err := store.ApplyEncoded(ctx, string(evt.Type()), evt.Payload())
//
// To integrate with projection.Projection, create a thin adapter:
//
//	type projectionAdapter struct{ store *metaengine.Store }
//	func (p *projectionAdapter) Handle(ctx context.Context, evt event.Event) error {
//	    return p.store.ApplyEncoded(ctx, string(evt.Type()), evt.Payload())
//	}
//
// The projectionadapter package already does exactly that, decoding nothing
// itself: prefer it over a hand-written adapter.
func (s *Store) ApplyEncoded(ctx context.Context, eventType string, payload []byte) error {
	encoded, err := encodePayload(payload, "")
	if err != nil {
		return fmt.Errorf("metaengine.Store.ApplyEncoded %s: %w", eventType, err)
	}

	return s.applyWithRecord(
		ctx,
		feedApplyEncoded,
		eventType,
		record.Record{Type: eventType},
		encoded,
	)
}

// ApplyEncodedRecord is ApplyEncoded with full Record context (ADR-0112):
// Record-aware folds (created via OnRecord) receive rec alongside the
// decoded payload, exactly like Store.ApplyRecord. rec.Type must name the
// event type the folds listen for. Non-Record-aware folds receive only the
// decoded payload, as usual.
//
// rec.Encoding, when known, selects the payload codec; an unknown stamp falls
// back to detection. JSON and CBOR are supported.
func (s *Store) ApplyEncodedRecord(ctx context.Context, rec record.Record, payload []byte) error {
	if rec.Type == "" {
		return errors.New(
			"metaengine.Store.ApplyEncodedRecord: Record.Type is empty — it must name the event type",
		)
	}

	encoded, err := encodePayload(payload, codecEncodingOf(rec.Encoding))
	if err != nil {
		return fmt.Errorf("metaengine.Store.ApplyEncodedRecord %s: %w", rec.Type, err)
	}

	return s.applyWithRecord(ctx, feedApplyEncodedRecord, rec.Type, rec, encoded)
}

// encodedPayload is the pipeline marker for an undecoded payload that is NOT
// plain JSON. Folds decode it with the codec that produced it and their own
// sample type.
//
// Schema-free transcoding to JSON is deliberately not used here: it cannot
// represent every Go value (time.Time encodes as an untagged CBOR epoch
// integer, which a JSON decode into time.Time rejects), so a transcoded
// payload would fail — or worse, silently change shape — at the fold.
type encodedPayload struct {
	raw      []byte
	encoding codec.Encoding
}

// encodePayload converts caller-supplied bytes into the pipeline marker for
// their codec. JSON keeps the historical jsontext.Value marker (decoded by
// encoding/json/v2 per fold); CBOR carries its encoding so each fold can decode
// the bytes with the codec that produced them.
func encodePayload(payload []byte, stamped codec.Encoding) (any, error) {
	if len(payload) == 0 {
		return nil, errorfamily.NewCorruption(
			"metaengine.encoded.empty_payload",
			"metaengine: encoded payload is empty",
		)
	}

	encoding := stamped
	if encoding == "" || encoding == codec.EncodingRaw {
		encoding = codec.AutoDetect(payload)
	}

	switch encoding {
	case codec.EncodingJSON:
		return rawJSON(payload), nil
	case codec.EncodingCBOR:
		return encodedPayload{raw: slices.Clone(payload), encoding: encoding}, nil
	default:
		return nil, errorfamily.NewCorruption(
			"metaengine.encoded.unknown_encoding",
			"metaengine: payload is neither JSON nor CBOR",
		)
	}
}

// codecEncodingOf converts a Record's compact stamp into the codec vocabulary.
// The zero stamp (EncodingUnknown) yields "", which encodePayload treats as
// "detect from the bytes".
func codecEncodingOf(encoding record.Encoding) codec.Encoding {
	switch encoding {
	case record.EncodingJSON:
		return codec.EncodingJSON
	case record.EncodingCBOR:
		return codec.EncodingCBOR
	default:
		return ""
	}
}

// rawJSON clones the caller's bytes into a jsontext.Value — the pipeline's
// marker for an undecoded JSON payload (applyFold decodes it per fold via
// the fold's sample). The clone protects the EventLog entry and any queued
// replication job from caller-side mutation after the call returns.
func rawJSON(payload []byte) jsontext.Value {
	return jsontext.Value(slices.Clone(payload))
}

// decodeRawFoldPayload decodes an undecoded payload into the fold's expected
// event type (via its sample) before invoke, so encoded applies replay
// identically on every dispatch path (primary folds, shadows, replays).
// Decoded payloads pass through unchanged — one type switch on the struct hot
// path. Folds whose sample is itself a byte slice consume the raw bytes
// undecoded.
func decodeRawFoldPayload(fold Fold, payload any) (any, error) {
	switch raw := payload.(type) {
	case jsontext.Value:
		return decodeJSONFoldPayload(fold, raw)
	case encodedPayload:
		return decodeEncodedFoldPayload(fold, raw)
	default:
		return payload, nil
	}
}

func decodeJSONFoldPayload(fold Fold, raw jsontext.Value) (any, error) {
	if isByteSample(fold.EventSample()) {
		return []byte(raw), nil
	}

	decoded, err := decodeFromSample(fold.EventSample(), raw)
	if err != nil {
		return nil, corruptionDecodeError(fold, err)
	}

	return decoded, nil
}

func decodeEncodedFoldPayload(fold Fold, payload encodedPayload) (any, error) {
	if isByteSample(fold.EventSample()) {
		return slices.Clone(payload.raw), nil
	}

	codecForPayload, err := codec.ForEncoding(payload.encoding)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err,
			"metaengine.apply.encoded_codec",
			"decode "+fold.EventType()+": no codec for encoding "+string(payload.encoding))
	}

	t := derefType(fold.EventSample())

	v := reflect.New(t)
	if err := codecForPayload.Decode(payload.raw, v.Interface()); err != nil {
		return nil, corruptionDecodeError(fold, err)
	}

	return v.Elem().Interface(), nil
}

// corruptionDecodeError classifies a fold payload decode failure. The payload
// bytes are immutable, so retrying cannot fix the failure: Corruption is the
// family that lets projection hosts dead-letter instead of crash-looping.
func corruptionDecodeError(fold Fold, err error) error {
	return errorfamily.WrapCorruption(err,
		"metaengine.apply.encoded_decode", "decode "+fold.EventType())
}

func isByteSample(sample any) bool {
	t := derefType(sample)

	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

func decodeFromSample(sample any, payload []byte) (any, error) {
	t := derefType(sample)

	v := reflect.New(t)
	if err := json.Unmarshal(payload, v.Interface()); err != nil {
		return nil, fmt.Errorf("json decode into %s: %w", t.Name(), err)
	}

	return v.Elem().Interface(), nil
}
