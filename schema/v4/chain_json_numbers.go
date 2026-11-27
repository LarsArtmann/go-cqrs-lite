package schema

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// exactIntLimit is the range in which every integer has an exact float64.
const exactIntLimit = 1 << 53

// decodeJSONFieldMap decodes a JSON payload field map without losing integers.
// Decoding numbers as float64 silently rounds every integer above 2^53 in
// fields the op never touches; json.Number keeps the literal.
func decodeJSONFieldMap(evt event.Event) (map[string]any, error) {
	var fields map[string]any
	if err := decodeExact(evt.Payload(), &fields); err != nil {
		return nil, wrapDecodeErr(evt, err)
	}

	if fields != nil {
		exactNumbers(fields)
	}

	return fields, nil
}

func decodeExact(payload []byte, fields *map[string]any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	return decoder.Decode(fields) //nolint:wrapcheck // caller wraps with event context
}

// exactNumbers restores float64 in place for numbers that were exact before
// (the documented field-map shape) and keeps larger integers as int64/uint64.
func exactNumbers(fields map[string]any) {
	for key, value := range fields {
		fields[key] = exactValue(value)
	}
}

func exactValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		return intOrFloat(typed)
	case map[string]any:
		exactNumbers(typed)

		return typed
	case []any:
		for i, item := range typed {
			typed[i] = exactValue(item)
		}

		return typed
	default:
		return value
	}
}

func intOrFloat(n json.Number) any {
	if i, err := n.Int64(); err == nil {
		if i > exactIntLimit || i < -exactIntLimit {
			return i
		}

		return float64(i)
	}

	if u, err := strconv.ParseUint(n.String(), 10, 64); err == nil {
		return u
	}

	f, _ := n.Float64() // non-integers; the literal came from a valid JSON number

	return f
}
