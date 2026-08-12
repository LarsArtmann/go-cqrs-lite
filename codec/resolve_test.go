package codec

import (
	"errors"
	"testing"
)

func TestForEncoding_JSON(t *testing.T) {
	t.Parallel()

	c, err := ForEncoding(EncodingJSON)
	if err != nil {
		t.Fatalf("ForEncoding(JSON): %v", err)
	}

	if c.Encoding() != EncodingJSON {
		t.Errorf("Encoding() = %q, want %q", c.Encoding(), EncodingJSON)
	}
}

func TestForEncoding_CBOR(t *testing.T) {
	t.Parallel()

	c, err := ForEncoding(EncodingCBOR)
	if err != nil {
		t.Fatalf("ForEncoding(CBOR): %v", err)
	}

	if c.Encoding() != EncodingCBOR {
		t.Errorf("Encoding() = %q, want %q", c.Encoding(), EncodingCBOR)
	}
}

func TestForEncoding_UnknownReturnsError(t *testing.T) {
	t.Parallel()

	cases := []Encoding{
		EncodingRaw,
		"encrypted",
		"msgpack",
		"",
	}

	for _, enc := range cases {
		c, err := ForEncoding(enc)
		if err == nil {
			t.Errorf("ForEncoding(%q) = %v, want error", enc, c)

			continue
		}

		if !errors.Is(err, ErrUnknownEncoding) {
			t.Errorf("ForEncoding(%q) err = %v, want ErrUnknownEncoding", enc, err)
		}
	}
}

func TestForEncoding_RoundTripsWithAutoDetect(t *testing.T) {
	t.Parallel()

	type payload struct{ Name string }

	for _, codec := range []Codec{JSONCodec{}, CBORCodec{}} {
		data, err := codec.Encode(payload{Name: "Alice"})
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		detected := AutoDetect(data)

		resolved, err := ForEncoding(detected)
		if err != nil {
			t.Fatalf("ForEncoding(%s): %v", detected, err)
		}

		if resolved.Encoding() != codec.Encoding() {
			t.Errorf(
				"ForEncoding(AutoDetect(data)) = %s, want %s",
				resolved.Encoding(),
				codec.Encoding(),
			)
		}

		var got payload
		if err := resolved.Decode(data, &got); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		if got.Name != "Alice" {
			t.Errorf("Name = %q, want %q", got.Name, "Alice")
		}
	}
}
