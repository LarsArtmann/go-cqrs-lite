package codec_test

import (
	"testing"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

func FuzzJSONCodec_Roundtrip(f *testing.F) {
	f.Add(`{"name":"Alice","age":30}`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`[]`)
	f.Add(`"hello"`)
	f.Add(`42`)
	f.Add(`true`)

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}

		c := codec.JSONCodec{}

		var decoded any
		if err := c.Decode([]byte(input), &decoded); err != nil {
			t.Skip()
		}

		encoded, err := c.Encode(decoded)
		if err != nil {
			t.Fatalf("Encode(%v): %v", decoded, err)
		}

		var redecoded any
		if err := c.Decode(encoded, &redecoded); err != nil {
			t.Fatalf("Decode(re-encoded): %v", err)
		}
	})
}

func FuzzRawCodec_Passthrough(f *testing.F) {
	f.Add([]byte(`{"raw":true}`))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff, 0xfe, 0x01})

	f.Fuzz(func(t *testing.T, input []byte) {
		c := codec.RawCodec{}

		got, err := c.Encode(input)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		var target []byte
		if err := c.Decode(got, &target); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		if len(input) == 0 && len(target) == 0 {
			return
		}

		if string(target) != string(input) {
			t.Errorf("roundtrip mismatch: got %q, want %q", target, input)
		}
	})
}
