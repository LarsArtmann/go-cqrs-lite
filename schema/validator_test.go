package schema

import (
	"encoding/json/v2"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type userCreatedPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func testEvent(t *testing.T, eventType string, payload []byte) event.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type(eventType), aggID, "User", 1, payload,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func TestValidator_ValidPayload_Passes(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	payload, err := json.Marshal(userCreatedPayload{Name: "Alice", Email: "alice@test.com"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	evt := testEvent(t, "user.created", payload)

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected valid payload to pass, got: %v", err)
	}
}

func TestValidator_InvalidPayload_Rejected(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	evt := testEvent(t, "user.created", []byte(`{"name": 123}`))

	err := v.Validate(evt)
	if err == nil {
		t.Fatal("expected rejection for malformed payload")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection error, got %T: %v", err, err)
	}
}

func TestValidator_MalformedJSON_Rejected(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	evt := testEvent(t, "user.created", []byte(`{broken`))

	err := v.Validate(evt)
	if err == nil {
		t.Fatal("expected rejection for malformed JSON")
	}
}

func TestValidator_UnregisteredType_LenientMode(t *testing.T) {
	t.Parallel()

	v := NewValidator()

	evt := testEvent(t, "unknown.event", []byte(`{"anything": true}`))

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected unregistered type to pass in lenient mode, got: %v", err)
	}
}

func TestValidator_UnregisteredType_StrictMode(t *testing.T) {
	t.Parallel()

	v := NewValidator(WithStrictMode())

	evt := testEvent(t, "unknown.event", []byte(`{}`))

	err := v.Validate(evt)
	if err == nil {
		t.Fatal("expected rejection for unregistered type in strict mode")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection error, got %T: %v", err, err)
	}
}

func TestValidator_CustomValidation(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterTypeWithValidator(v, "user.created", func(u userCreatedPayload) error {
		if u.Name == "" {
			return errors.New("name is required")
		}

		return nil
	})

	tests := []struct {
		name    string
		payload userCreatedPayload
		wantErr bool
	}{
		{"valid", userCreatedPayload{Name: "Bob", Email: "bob@test.com"}, false},
		{"empty name", userCreatedPayload{Name: "", Email: "x@test.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			evt := testEvent(t, "user.created", payload)

			err = v.Validate(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidator_EmptyPayload_Passes(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	evt := testEvent(t, "user.created", nil)

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected empty payload to pass, got: %v", err)
	}
}

func TestValidator_RegisteredTypes(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")
	RegisterType[userCreatedPayload](v, "user.updated")

	types := v.RegisteredTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 registered types, got %d", len(types))
	}
}

func TestValidator_WithCustomCodec(t *testing.T) {
	t.Parallel()

	v := NewValidator(WithCodec(codec.JSONCodec{}))
	RegisterType[userCreatedPayload](v, "user.created")

	payload, err := json.Marshal(userCreatedPayload{Name: "Alice"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	evt := testEvent(t, "user.created", payload)

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected valid payload with custom codec to pass, got: %v", err)
	}
}

func TestValidator_WithCodec_CBOR(t *testing.T) {
	t.Parallel()

	v := NewValidator(WithCodec(codec.CBORCodec{}))
	RegisterType[userCreatedPayload](v, "user.created")

	aggID := id.NewAggregateID()
	evt, err := event.New(
		event.Type("user.created"), aggID, "User", 1,
		userCreatedPayload{Name: "Alice", Email: "alice@test.com"},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("New with CBOR codec: %v", err)
	}

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected CBOR payload via WithCodec to pass, got: %v", err)
	}
}

func TestValidator_CBOREncoding_AutoDetected(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	cborCodec := codec.CBORCodec{}

	aggID := id.NewAggregateID()
	evt, err := event.New(
		event.Type("user.created"), aggID, "User", 1,
		userCreatedPayload{Name: "Alice", Email: "alice@test.com"},
		event.WithCodec(cborCodec),
	)
	if err != nil {
		t.Fatalf("New with CBOR codec: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("event encoding = %s, want %s", evt.Encoding(), codec.EncodingCBOR)
	}

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected CBOR payload to validate via auto-detection, got: %v", err)
	}
}

func TestValidator_CBOREncoding_Invalid_Rejected(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"), aggID, "User", 1, []byte{0xa0},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = v.Validate(evt)
	if err == nil {
		t.Fatal("expected rejection for invalid CBOR payload")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection, got %T: %v", err, err)
	}
}

func TestValidator_EncryptedEncoding_RejectedGracefully(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	// Simulate an encrypted event: random-looking ciphertext stamped "encrypted".
	// The validator has no decoder for "encrypted" — it should fall back to the
	// default JSON decoder, which will reject the ciphertext as malformed JSON,
	// producing a clean Rejection error (not a panic).
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"), aggID, "User", 1,
		[]byte{0x72, 0x4a, 0x8f, 0x3b, 0xc1, 0xe9, 0xd0, 0x5a},
		event.WithEncoding(codec.Encoding("encrypted")),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = v.Validate(evt)
	if err == nil {
		t.Fatal("expected rejection for encrypted payload with no matching decoder")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection, got %T: %v", err, err)
	}
}

func TestValidator_UnknownEncoding_FallsBackToJSON(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	RegisterType[userCreatedPayload](v, "user.created")

	// An unknown encoding falls back to the default JSON decoder.
	// A valid JSON payload should pass.
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"), aggID, "User", 1,
		[]byte(`{"name":"Alice","email":"alice@test.com"}`),
		event.WithEncoding(codec.Encoding("msgpack")),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = v.Validate(evt)
	if err != nil {
		t.Fatalf("expected unknown encoding to fall back to JSON and pass, got: %v", err)
	}
}

func TestValidator_EncryptedEncoding_WithCustomDecoder(t *testing.T) {
	t.Parallel()

	// If a consumer registers a decoder for the "encrypted" encoding (e.g.
	// wrapping a decrypt + unmarshal pipeline), the validator should use it.
	v := NewValidator(WithDecoder(codec.Encoding("encrypted"), func(data []byte, target any) error {
		return json.Unmarshal([]byte(`{"name":"Alice","email":"alice@test.com"}`), target)
	}))
	RegisterType[userCreatedPayload](v, "user.created")

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"), aggID, "User", 1,
		[]byte{0xde, 0xad, 0xbe, 0xef},
		event.WithEncoding(codec.Encoding("encrypted")),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := v.Validate(evt); err != nil {
		t.Fatalf("expected custom encrypted decoder to validate, got: %v", err)
	}
}
