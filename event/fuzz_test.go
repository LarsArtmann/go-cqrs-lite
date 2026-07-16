package event_test

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func FuzzParseSource(f *testing.F) {
	f.Add("api")
	f.Add("")
	f.Add("scheduler")
	f.Add("  spaces  ")
	f.Add("source-with-unicode-\u00e9")

	f.Fuzz(func(t *testing.T, input string) {
		src, err := event.ParseSource(input)

		trimmed := trimSpaces(input)
		if trimmed == "" {
			if err == nil {
				t.Error("expected error for empty source")
			}

			return
		}

		if err != nil {
			t.Errorf("unexpected error for %q: %v", input, err)
		}

		if src.String() != trimmed {
			t.Errorf("roundtrip mismatch: got %q, want %q", src.String(), trimmed)
		}
	})
}

func FuzzParseIPAddress(f *testing.F) {
	f.Add("192.168.1.1")
	f.Add("")
	f.Add("::1")
	f.Add("not-an-ip")
	f.Add("999.999.999.999")
	f.Add("2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	f.Fuzz(func(_ *testing.T, input string) {
		_, _ = event.ParseIPAddress(input)
	})
}

func FuzzParseVersion(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(1000000))

	f.Fuzz(func(t *testing.T, v uint64) {
		ver, err := event.ParseVersion(v)
		if err != nil {
			t.Errorf("unexpected error for %d: %v", v, err)
		}

		if ver.UInt64() != v {
			t.Errorf("roundtrip mismatch: got %d, want %d", ver.UInt64(), v)
		}
	})
}

func trimSpaces(s string) string {
	return strings.TrimSpace(s)
}

func FuzzNewEvent(f *testing.F) {
	f.Add("user.created", "User", int64(1), int64(1), `{"name":"test"}`)
	f.Add("", "", int64(0), int64(0), "")
	f.Add("a", "B", int64(1), int64(1), "{}")
	f.Add(
		strings.Repeat("x", 256),
		strings.Repeat("y", 256),
		int64(999999),
		int64(1),
		`{"data":"`+strings.Repeat("A", 10000)+`"}`,
	)

	f.Fuzz(
		func(t *testing.T, eventType, aggType string, version, schemaVersion int64, payload string) {
			aggID := id.NewAggregateID()
			evt, err := event.NewEvent(
				event.Type(eventType), aggID, id.AggregateType(aggType),
				event.Version(int(version)), []byte(payload),
				event.WithSchemaVersion(event.SchemaVersion(int(schemaVersion))),
			)
			if err != nil {
				return
			}

			if evt.AggregateID() != aggID {
				t.Error("aggregate ID mismatch")
			}
		},
	)
}

func FuzzDecodePayload(f *testing.F) {
	type testPayload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	f.Add(`{"name":"Alice","email":"alice@example.com"}`)
	f.Add(`{}`)
	f.Add(``)
	f.Add(`not json at all`)
	f.Add(`{"name":"` + strings.Repeat("A", 10000) + `"}`)
	f.Add(`{"name":null}`)

	f.Fuzz(func(t *testing.T, payloadJSON string) {
		aggID := id.NewAggregateID()

		evt, err := event.NewEvent(
			event.Type("test"), aggID, id.AggregateType("Test"),
			event.Version(1), jsontext.Value(payloadJSON),
		)
		if err != nil {
			return
		}

		_, _ = event.DecodePayload[testPayload](evt, codec.JSONCodec{})
	})
}
