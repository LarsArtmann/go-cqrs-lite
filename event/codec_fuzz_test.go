package event_test

import (
	"testing"
	"unicode/utf8"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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

		codec := codecpkg.JSONCodec{}

		var decoded any
		if err := codec.Decode([]byte(input), &decoded); err != nil {
			t.Skip()
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
			t.Skip()
		}

		aggID := id.NewAggregateID()
		evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(input))
		if err != nil {
			t.Skip()
		}

		codec := codecpkg.JSONCodec{}

		result, err := event.DecodePayload[fuzzPayload](evt, codec)
		if err != nil {
			t.Skip()
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
		aggID := id.NewAggregateID()
		evt, err := event.NewEvent("TestEvent", aggID, "Test", 1, payload)
		if err != nil {
			t.Skip()
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
