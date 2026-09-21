package benchkit

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"time"
)

// durationMarshalers serializes time.Duration as nanoseconds (int64)
// because JSON v2 has no default representation for time.Duration.
var durationMarshalers = json.MarshalFunc(
	func(d time.Duration) ([]byte, error) {
		return []byte(strconv.FormatInt(d.Nanoseconds(), 10)), nil
	},
)

// durationUnmarshalers deserializes time.Duration from nanoseconds (int64),
// enabling JSON round-trip (WriteJSON → json.Unmarshal with jsonOpts).
var durationUnmarshalers = json.UnmarshalFunc(
	func(b []byte, t *time.Duration) error {
		n, err := strconv.ParseInt(string(b), 10, 64)
		if err != nil {
			return fmt.Errorf("parse duration nanoseconds: %w", err)
		}

		*t = time.Duration(n)

		return nil
	},
)

// jsonOpts are the default JSON encoding options: indented output
// with time.Duration serialized/deserialized as nanoseconds.
var jsonOpts = json.JoinOptions(
	jsontext.WithIndent("  "),
	json.WithMarshalers(durationMarshalers),
	json.WithUnmarshalers(durationUnmarshalers),
)

// WriteJSON serializes a result as indented JSON.
func WriteJSON(w io.Writer, r *Result) error {
	return json.MarshalWrite(w, r, jsonOpts)
}

// writeJSONAny serializes any value as indented JSON using the standard options.
func writeJSONAny(w io.Writer, v any) error {
	return json.MarshalWrite(w, v, jsonOpts)
}
