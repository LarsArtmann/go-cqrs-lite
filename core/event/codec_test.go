package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func TestJSONCodec_Encode(t *testing.T) {
	t.Parallel()

	codec := event.JSONCodec{}

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

	codec := event.JSONCodec{}

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

	codec := event.JSONCodec{}

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
	if err := codec.Decode(data, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded != original {
		t.Errorf("roundtrip: got %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_Encode_Nil(t *testing.T) {
	t.Parallel()

	codec := event.JSONCodec{}

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

	codec := event.JSONCodec{}

	var v any

	err := codec.Decode([]byte(`{invalid`), &v)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCodecInterface(t *testing.T) {
	t.Parallel()

	var _ event.Codec = event.JSONCodec{}
}
