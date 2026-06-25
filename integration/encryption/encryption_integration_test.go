package encryption_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/encryption/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
)

func TestSignAndEncryptFullFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	signingKey := []byte("signing-secret-key-thirty-two-by!")
	hmacSigner, _ := signing.NewHMAC(signingKey)

	encryptKey := make([]byte, 32)
	for i := range encryptKey {
		encryptKey[i] = byte(i)
	}

	xchacha, _ := encryption.NewXChaCha20Poly1305(encryptKey)

	_ = bus.UsePublish(signing.SignMiddleware(hmacSigner))
	_ = bus.UsePublish(
		encryption.EncryptMiddleware(
			xchacha,
			encryption.WithMiddlewareKeyID(encryption.KeyID("key-v1")),
		),
	)

	_ = bus.Use(encryption.DecryptMiddleware(xchacha))
	_ = bus.Use(signing.VerifyMiddleware(hmacSigner))

	var received []event.Event

	if err := bus.Subscribe("user.created", func(_ context.Context, evt event.Event) error {
		received = append(received, evt)

		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		"user.created",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(ctx, evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("expected 1 received event, got %d", len(received))
	}

	got := received[0]
	if string(got.Payload()) != `{"name":"Alice"}` {
		t.Fatalf("payload mismatch: got %q", string(got.Payload()))
	}

	if got.Type() != "user.created" {
		t.Fatalf("type mismatch: got %q", got.Type())
	}

	if got.AggregateID() != aggID {
		t.Fatalf("aggregate ID mismatch")
	}

	if encryption.HasEncryption(got) {
		t.Fatal("decrypted event should not have encryption metadata")
	}

	if !signing.HasSignature(got) {
		t.Fatal("verified event should still carry signature (verify passes events through)")
	}
}

func TestEncryptThenSignCodecRoundtrip(t *testing.T) {
	t.Parallel()

	encryptKey := make([]byte, 32)
	for i := range encryptKey {
		encryptKey[i] = byte(i)
	}

	xchacha, _ := encryption.NewXChaCha20Poly1305(encryptKey)

	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	original := payload{Name: "Bob", Age: 30}

	codec := encryption.NewCodec(codec.JSONCodec{}, xchacha)

	encoded, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if len(encoded) == 0 {
		t.Fatal("encoded should not be empty")
	}

	if codec.Encoding() != encryption.EncryptionEncoding {
		t.Fatalf("encoding mismatch: got %q", codec.Encoding())
	}

	var decoded payload
	if err := codec.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Name != original.Name {
		t.Fatalf("name mismatch: got %q, want %q", decoded.Name, original.Name)
	}

	if decoded.Age != original.Age {
		t.Fatalf("age mismatch: got %d, want %d", decoded.Age, original.Age)
	}
}

func TestEncryptMiddleware_DetectsAlgorithm_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	aes, _ := encryption.NewAES256GCM(key)

	_ = bus.UsePublish(
		encryption.EncryptMiddleware(
			aes,
			encryption.WithMiddlewareKeyID(encryption.KeyID("aes-key-1")),
		),
	)

	var intercepted []event.Event

	if err := bus.Subscribe("test.event", func(_ context.Context, evt event.Event) error {
		intercepted = append(intercepted, evt)

		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	evt, _ := event.NewEvent(
		"test.event",
		id.NewAggregateID(),
		"Test",
		1,
		[]byte(`{"data":"test"}`),
	)
	_ = bus.Publish(ctx, evt)

	time.Sleep(50 * time.Millisecond)

	if len(intercepted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(intercepted))
	}

	alg, err := encryption.ExtractAlgorithm(intercepted[0])
	if err != nil {
		t.Fatalf("ExtractAlgorithm: %v", err)
	}

	if alg != encryption.AES256GCM {
		t.Fatalf("expected AES256GCM, got %s", alg)
	}

	keyID, _ := encryption.ExtractKeyID(intercepted[0])
	if keyID != "aes-key-1" {
		t.Fatalf("expected key-id 'aes-key-1', got %q", keyID)
	}
}
