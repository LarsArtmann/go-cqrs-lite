package schema

import (
	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// The payload layer of chain execution: decode/re-encode field maps with the
// event's own encoding stamp, and rebuild events with preserved identity.

func wrapDecodeErr(evt event.Event, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.op_decode_failed",
		"decode "+string(evt.Encoding())+" payload of "+string(evt.Type())+
			" v"+evt.SchemaVersion().String(),
	)
}

func wrapTransformErr(evt event.Event, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.op_transform_failed",
		"transform "+string(evt.Type())+" v"+evt.SchemaVersion().String(),
	)
}

// decodeFieldMap decodes the payload with the codec the event's Encoding()
// stamp selects — self-describing events, mixed JSON/CBOR streams included.
//
// Nested maps are normalized to map[string]any so Transform/field ops see one
// shape regardless of encoding: fxamacker/cbor decodes maps into
// map[any]any when the target is any, while encoding/json yields
// map[string]any — without normalization a nested-object Transform works on
// JSON events and silently no-ops on CBOR events. Maps with non-string keys
// are outside the map[string]any field-map contract and are left as decoded.
func decodeFieldMap(evt event.Event) (map[string]any, error) {
	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	var fields map[string]any
	if err := codecFor.Decode(evt.Payload(), &fields); err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	if fields == nil {
		fields = make(map[string]any)
	}

	normalizeStringKeyMaps(fields)

	return fields, nil
}

// normalizeStringKeyMaps rewrites every nested map[any]any whose keys are all
// strings into map[string]any, in place. Slices are walked recursively.
// A map with any non-string key is returned unchanged (all-or-nothing per
// map) — it cannot be represented in the field-map contract.
func normalizeStringKeyMaps(fields map[string]any) {
	for key, value := range fields {
		fields[key] = normalizeValue(value)
	}
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalizeStringKeyMaps(typed)

		return typed
	case map[any]any:
		normalized := make(map[string]any, len(typed))

		for anyKey, nested := range typed {
			strKey, isString := anyKey.(string)
			if !isString {
				return typed // non-string key: outside the contract, keep as-is
			}

			normalized[strKey] = normalizeValue(nested)
		}

		return normalized
	case []any:
		for i, item := range typed {
			typed[i] = normalizeValue(item)
		}

		return typed
	default:
		return value
	}
}

func encodeFieldMap(evt event.Event, fields map[string]any) ([]byte, error) {
	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	payload, err := codecFor.Encode(fields)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err, "schema.op_encode_failed",
			"encode payload of "+string(evt.Type()),
		)
	}

	return payload, nil
}

// rebuild constructs the upcasted event for 1:1 ops: fresh instance,
// preserved identity (event ID, stream, position, timestamp, metadata,
// encoding), new type and schema version. payload is already encoded bytes.
func rebuild(
	evt event.Event,
	payload []byte,
	eventType event.Type,
	schemaVersion event.SchemaVersion,
) (event.Event, error) {
	upcasted, err := event.New(
		eventType,
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		payload,
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithMetadata(evt.Metadata()),
		event.WithEncoding(evt.Encoding()),
		event.WithSchemaVersion(schemaVersion),
	)
	if err != nil {
		return nil, wrapRebuildErr(evt, eventType, err)
	}

	return upcasted, nil
}

func wrapRebuildErr(evt event.Event, eventType event.Type, err error) error {
	return errorfamily.WrapCorruption(
		err, "schema.rebuild_failed",
		"rebuild "+string(eventType)+" v"+evt.SchemaVersion().String()+
			" (source "+string(evt.Type())+")",
	)
}

// encodeValue encodes a Split payload function's return value with the
// codec matching the SOURCE event's encoding, so payload bytes and the
// encoding stamp always agree ([]byte payloads pass through untouched).
func encodeValue(evt event.Event, payload any) ([]byte, error) {
	if raw, isBytes := payload.([]byte); isBytes {
		return raw, nil
	}

	codecFor, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	encoded, err := codecFor.Encode(payload)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err, "schema.op_encode_failed",
			"encode payload of "+string(evt.Type()),
		)
	}

	return encoded, nil
}
