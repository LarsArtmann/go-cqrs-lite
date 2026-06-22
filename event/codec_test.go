package event_test

import (
	"errors"
	"fmt"
	"testing"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3/idtest"
)

func TestJSONCodec_Encode(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}

	got, err := codec.Encode(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	want := `{"key":"value"}`
	if string(got) != want {
		t.Errorf("Encode = %q, want %q", string(got), want)
	}
}

func TestJSONCodec_Decode(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}

	var got struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	err := codec.Decode([]byte(`{"name":"Alice","age":30}`), &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", got.Name)
	}

	if got.Age != 30 {
		t.Errorf("Age = %d, want 30", got.Age)
	}
}

func TestJSONCodec_Roundtrip(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}

	type userCreated struct {
		UserID string `json:"userId"`
		Name   string `json:"name"`
		Email  string `json:"email"`
	}

	original := userCreated{
		UserID: "user-123",
		Name:   "Bob",
		Email:  "bob@example.com",
	}

	data, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var decoded userCreated

	err = codec.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded != original {
		t.Errorf("roundtrip: got %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_Encode_Nil(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}

	got, err := codec.Encode(nil)
	if err != nil {
		t.Fatalf("Encode nil: %v", err)
	}

	if string(got) != "null" {
		t.Errorf("Encode nil = %q, want null", string(got))
	}
}

func TestJSONCodec_Decode_InvalidJSON(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}

	var v any

	err := codec.Decode([]byte(`{invalid`), &v)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCodecInterface(t *testing.T) {
	t.Parallel()

	var _ codecpkg.Codec = codecpkg.JSONCodec{}
}

func TestDecodePayload(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}
	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	type userPayload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	evt, err := event.NewEvent(
		"UserCreated",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice","email":"alice@example.com"}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	result, err := event.DecodePayload[userPayload](evt, codec)
	if err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", result.Name)
	}

	if result.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", result.Email)
	}
}

func TestDecodePayload_EmptyPayload(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}
	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserDeleted", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	result, err := event.DecodePayload[struct{}](evt, codec)
	if err != nil {
		t.Fatalf("DecodePayload empty: %v", err)
	}

	_ = result
}

func TestDecodePayload_InvalidJSON(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}
	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{broken`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	_, err = event.DecodePayload[struct{ Name string }](evt, codec)
	if err == nil {
		t.Error("expected error for invalid JSON payload")
	}
}

type failingCodec struct{}

func (failingCodec) Encoding() codecpkg.Encoding  { return codecpkg.EncodingJSON }
func (failingCodec) Encode(_ any) ([]byte, error) { return nil, errors.New("encode fail") }
func (failingCodec) Decode(_ []byte, _ any) error { return errors.New("decode fail") }

func TestDecodePayload_CodecError(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	_, err = event.DecodePayload[struct{}](evt, failingCodec{})
	if err == nil {
		t.Error("expected error from failing codec")
	}
}

func TestDecodePayloads(t *testing.T) {
	t.Parallel()

	codec := codecpkg.JSONCodec{}
	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	type userPayload struct {
		Name string `json:"name"`
	}

	events := make([]event.Event, 3)
	for i := range 3 {
		evt, err := event.NewEvent(
			"UserCreated", aggID, "User", event.Version(i+1),
			fmt.Appendf(nil, `{"name":"User%d"}`, i),
		)
		if err != nil {
			t.Fatalf("NewEvent[%d]: %v", i, err)
		}

		events[i] = evt
	}

	results, err := event.DecodePayloads[userPayload](events, codec)
	if err != nil {
		t.Fatalf("DecodePayloads: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("len = %d, want 3", len(results))
	}

	names := []string{"User0", "User1", "User2"}
	for i, want := range names {
		if results[i].Name != want {
			t.Errorf("results[%d].Name = %q, want %q", i, results[i].Name, want)
		}
	}
}

func TestDecodePayloads_ErrorStopsAtFirst(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	goodEvt, _ := event.NewEvent("Good", aggID, "User", 1, []byte(`{}`))
	badEvt, _ := event.NewEvent("Bad", aggID, "User", 2, []byte(`{broken`))

	_, err := event.DecodePayloads[struct{ Name string }](
		[]event.Event{goodEvt, badEvt},
		codecpkg.JSONCodec{},
	)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecodePayload_EncodingMismatch(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent(
		"UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`),
		event.WithEncoding(codecpkg.Encoding("protobuf")),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	_, err = event.DecodePayload[struct{ Name string }](evt, codecpkg.JSONCodec{})
	if err == nil {
		t.Fatal("expected error for encoding mismatch")
	}
}

func TestDecodePayload_CBORCodec(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	payload := struct{ Name string }{Name: "Alice"}

	evt, err := event.New(
		"UserCreated", aggID, "User", 1, payload,
		event.WithCodec(codecpkg.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if evt.Encoding() != codecpkg.EncodingCBOR {
		t.Errorf("Encoding() = %q, want %q", evt.Encoding(), codecpkg.EncodingCBOR)
	}

	result, err := event.DecodePayload[struct{ Name string }](evt, codecpkg.CBORCodec{})
	if err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", result.Name)
	}
}

func TestDecodePayload_EncodingMatch(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent(
		"UserCreated", aggID, "User", 1, []byte(`{"name":"Alice"}`),
		event.WithEncoding(codecpkg.EncodingJSON),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	result, err := event.DecodePayload[struct{ Name string }](evt, codecpkg.JSONCodec{})
	if err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", result.Name)
	}
}

func TestEvent_Encoding_DefaultIsJSON(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if evt.Encoding() != codecpkg.EncodingJSON {
		t.Errorf("Encoding() = %q, want %q", evt.Encoding(), codecpkg.EncodingJSON)
	}
}

func TestPayloadReadOnly_ReturnsInternalReference(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"name":"Alice"}`)
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, payload)
	if err != nil {
		t.Fatal(err)
	}

	readOnly := event.PayloadReadOnly(evt)

	if string(readOnly) != string(payload) {
		t.Errorf("PayloadReadOnly = %q, want %q", readOnly, payload)
	}
}
