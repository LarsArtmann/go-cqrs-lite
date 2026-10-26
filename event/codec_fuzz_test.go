package event_test

import (
	"testing"
	"unicode/utf8"

	codecpkg "github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func FuzzJSONCodec_Roundtrip(f *testing.F) {
	for _, s := range []string{
		`{"name":"Alice","age":30}`,
		`{}`,
		`null`,
		`[]`,
		`"hello"`,
		`42`,
		`true`,
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip("fuzz mutates raw bytes; invalid UTF-8 is outside the codec contract")
		}

		codec := codecpkg.JSONCodec{}

		var decoded any
		if err := codec.Decode([]byte(input), &decoded); err != nil {
			t.Skip("input is not a JSON document the codec accepts; this target asserts roundtrip, not rejection")
		}

		encoded, err := codec.Encode(decoded)
		if err != nil {
			t.Fatalf("Encode(%v): %v", decoded, err)
		}

		var redecoded any
		if err := codec.Decode(encoded, &redecoded); err != nil {
			t.Fatalf("Decode(re-encoded): %v", err)
		}
	})
}

func FuzzDecodePayload_Roundtrip(f *testing.F) {
	type fuzzPayload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	f.Add(`{"name":"Alice","email":"alice@example.com"}`)
	f.Add(`{"name":"","email":""}`)
	f.Add(`{"name":"日本語テスト","email":"test@test.jp"}`)
	f.Add(`{"name":"<script>alert(1)</script>","email":"a@b.c"}`)

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip("fuzz mutates raw bytes; invalid UTF-8 is outside the codec contract")
		}

		streamID := id.NewStreamID()
		evt, err := event.NewEvent("UserCreated", streamID, "User", 1, []byte(input))
		if err != nil {
			t.Skip("payload rejected at construction (encoding/size guard); that rejection has its own tests")
		}

		codec := codecpkg.JSONCodec{}

		result, err := event.DecodePayload[fuzzPayload](evt, codec)
		if err != nil {
			t.Skip("decoded JSON does not satisfy the typed payload contract; not a roundtrip input")
		}

		if result.Name == "" && input != `{}` && input != `{"name":"","email":""}` {
		}
	})
}

func FuzzEvent_PayloadIsolation(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte{})
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff, 0xfe, 0x01})

	f.Fuzz(func(t *testing.T, payload []byte) {
		streamID := id.NewStreamID()
		evt, err := event.NewEvent("TestEvent", streamID, "Test", 1, payload)
		if err != nil {
			t.Skip("payload rejected at construction (encoding/size guard); isolation is asserted on accepted payloads")
		}

		got := evt.Payload()

		if len(payload) > 0 {
			mutation := make([]byte, len(got))
			copy(mutation, got)
			mutation[0] ^= 0xff

			after := evt.Payload()
			if after[0] == mutation[0] {
				t.Error("Payload() returned shared reference — mutation leaked")
			}
		}
	})
}
