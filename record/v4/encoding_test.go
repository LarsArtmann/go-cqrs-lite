package record_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestEncoding_String(t *testing.T) {
	t.Parallel()

	cases := map[record.Encoding]string{
		record.EncodingJSON:    "json",
		record.EncodingCBOR:    "cbor",
		record.EncodingUnknown: "",
		record.Encoding(99):    "",
	}

	for enc, want := range cases {
		if got := enc.String(); got != want {
			t.Errorf("Encoding(%d).String() = %q, want %q", uint8(enc), got, want)
		}
	}
}

func TestParseEncoding(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]record.Encoding{
		"json": record.EncodingJSON,
		"cbor": record.EncodingCBOR,
	} {
		got, err := record.ParseEncoding(name)
		if err != nil {
			t.Fatalf("ParseEncoding(%q): %v", name, err)
		}

		if got != want {
			t.Errorf("ParseEncoding(%q) = %v, want %v", name, got, want)
		}
	}

	got, err := record.ParseEncoding("protobuf")
	if !errors.Is(err, record.ErrUnknownEncoding) {
		t.Errorf("unknown name err = %v, want ErrUnknownEncoding", err)
	}

	if got != record.EncodingUnknown {
		t.Errorf("unknown name = %v, want EncodingUnknown", got)
	}
}

func TestEncoding_RoundTrip(t *testing.T) {
	t.Parallel()

	for _, enc := range []record.Encoding{record.EncodingJSON, record.EncodingCBOR} {
		back, err := record.ParseEncoding(enc.String())
		if err != nil {
			t.Fatalf("ParseEncoding(%q): %v", enc.String(), err)
		}

		if back != enc {
			t.Errorf("round trip %v -> %q -> %v lost the stamp", enc, enc.String(), back)
		}
	}
}

func TestEncoding_ZeroValueIsUnknown(t *testing.T) {
	t.Parallel()

	var rec record.Record
	if rec.Encoding != record.EncodingUnknown {
		t.Error("zero-value Record must carry EncodingUnknown")
	}

	if rec.Encoding.String() != "" {
		t.Errorf("EncodingUnknown.String() = %q, want empty", rec.Encoding.String())
	}
}
