package schema_test

import (
	"math"
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// Field ops re-encode the whole payload: an integer in a field the op never
// touches must survive byte-for-byte, in JSON as well as CBOR.
func TestFieldOpsPreserveLargeIntegers(t *testing.T) {
	t.Parallel()

	chain, err := schema.Compile(schema.RenameField("payment.made", 1, "ref", "reference"))
	if err != nil {
		t.Fatal(err)
	}

	transform := chain.SourceTransform()

	for _, enc := range []codec.Encoding{codec.EncodingJSON, codec.EncodingCBOR} {
		t.Run(string(enc), func(t *testing.T) {
			t.Parallel()

			out, err := transform([]event.Event{encodeForChain(t, enc)})
			if err != nil {
				t.Fatal(err)
			}

			got := asJSONText(t, enc, out[0].Payload())

			for _, want := range []string{
				`"cents":9007199254740993`, `-9007199254740993`, `"huge":18446744073709551615`,
				`9007199254740994`, `"reference":"inv-1"`, `"ratio":0.5`,
			} {
				if !strings.Contains(got, want) {
					t.Errorf("payload lost %s: %s", want, got)
				}
			}
		})
	}
}

// encodeForChain builds the same payload in the given encoding with native
// integer types, so the input never passes through float64.
func encodeForChain(t *testing.T, enc codec.Encoding) event.Event {
	t.Helper()

	fields := map[string]any{
		"ref":      "inv-1",
		"cents":    int64(9007199254740993),
		"negative": int64(-9007199254740993),
		"huge":     uint64(math.MaxUint64),
		"nested":   map[string]any{"ids": []any{int64(9007199254740994), int64(7)}},
		"ratio":    0.5,
	}

	codecFor, err := codec.ForEncoding(enc)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := codecFor.Encode(fields)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := event.NewEvent(
		"payment.made", id.NewStreamID(), "Payment", 1, encoded, event.WithEncoding(enc))
	if err != nil {
		t.Fatal(err)
	}

	return evt
}

// asJSONText renders a payload as JSON text without going through float64,
// so the test itself cannot round the integers it asserts on.
func asJSONText(t *testing.T, enc codec.Encoding, payload []byte) string {
	t.Helper()

	if enc == codec.EncodingJSON {
		return string(payload)
	}

	return string(transcode(t, enc, codec.EncodingJSON, payload))
}

func transcode(t *testing.T, from, to codec.Encoding, payload []byte) []byte {
	t.Helper()

	source, err := codec.ForEncoding(from)
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]any
	if err := source.Decode(payload, &fields); err != nil {
		t.Fatal(err)
	}

	target, err := codec.ForEncoding(to)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := target.Encode(fields)
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}
