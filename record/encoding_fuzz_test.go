package record_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// The Encoding stamp is the smallest type in the library and the easiest to
// get wrong: it sits on every Record, and its zero value carries a MEANING
// ("no codec stamp: absent, opaque, or envelope-wrapped") rather than being an
// unused default. These tests pin the two properties that meaning depends on.

// TestParseEncodingInvariants sweeps the whole Encoding value space (it is a
// uint8, so the space is finite) instead of sampling a few cases:
//
//   - String() only ever returns the canonical names, or "" for a non-canonical
//     value (a Record must never carry "Encoding(7)" into a payload header);
//   - every canonical encoding round-trips through ParseEncoding;
//   - EncodingUnknown renders as "" and is NOT parseable — "" means "no stamp",
//     so accepting it as a name would erase the distinction between an opaque
//     payload and a JSON one.
func TestParseEncodingInvariants(t *testing.T) {
	t.Parallel()

	canonical := map[record.Encoding]string{
		record.EncodingJSON: "json",
		record.EncodingCBOR: "cbor",
	}

	for raw := 0; raw <= 255; raw++ {
		enc := record.Encoding(raw)
		name := enc.String()

		wantName, isCanonical := canonical[enc]
		switch {
		case isCanonical && name != wantName:
			t.Fatalf("Encoding(%d).String() = %q, want %q", raw, name, wantName)
		case !isCanonical && enc != record.EncodingUnknown && name != "":
			t.Fatalf("Encoding(%d).String() = %q, want \"\" for a non-canonical value", raw, name)
		}

		if enc == record.EncodingUnknown {
			if name != "" {
				t.Fatalf("EncodingUnknown.String() = %q, want \"\"", name)
			}
			if _, err := record.ParseEncoding(name); !errors.Is(err, record.ErrUnknownEncoding) {
				t.Fatalf("ParseEncoding(\"\") error = %v, want ErrUnknownEncoding (the empty name is not a codec)", err)
			}

			continue
		}

		got, err := record.ParseEncoding(name)
		if err != nil {
			t.Fatalf("ParseEncoding(%q) failed for canonical name: %v", name, err)
		}
		if got != enc {
			t.Fatalf("ParseEncoding(%q) = %d, want %d", name, got, enc)
		}
	}
}

// FuzzParseEncodingTotality asserts ParseEncoding is total for arbitrary input:
// it never panics, any successful parse round-trips through String(), and
// anything outside the vocabulary fails with ErrUnknownEncoding rather than
// silently returning a wrong encoding. A silent mismatch would stamp the wrong
// codec on a payload, which surfaces much later as a decode failure.
func FuzzParseEncodingTotality(f *testing.F) {
	for _, seed := range []string{"json", "cbor", "", "JSON", "Cbor", "json ", "\x00", "application/json"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		enc, err := record.ParseEncoding(name)
		if err != nil {
			if !errors.Is(err, record.ErrUnknownEncoding) {
				t.Fatalf("ParseEncoding(%q) error = %v, want ErrUnknownEncoding", name, err)
			}
			if enc != record.EncodingUnknown {
				t.Fatalf("ParseEncoding(%q) returned %d with an error, want EncodingUnknown", name, enc)
			}

			return
		}

		if enc == record.EncodingUnknown {
			t.Fatalf("ParseEncoding(%q) succeeded with the zero encoding", name)
		}
		if got := enc.String(); got != name {
			t.Fatalf("ParseEncoding(%q) round-trips to %q", name, got)
		}
	})
}
