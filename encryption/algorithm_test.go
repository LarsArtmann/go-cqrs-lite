package encryption

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestAttachEncryption_WithAlgorithm(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	evt := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	ct, err := enc.Encrypt([]byte(`{"data":"hello"}`))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	attached, err := AttachEncryption(evt, ct, func(c *attachConfig) {
		c.algorithm = AES256GCM
	})
	if err != nil {
		t.Fatalf("AttachEncryption: %v", err)
	}

	alg, err := ExtractAlgorithm(attached)
	if err != nil {
		t.Fatalf("ExtractAlgorithm: %v", err)
	}

	if alg != AES256GCM {
		t.Fatalf("expected algorithm %s, got %s", AES256GCM, alg)
	}
}

func TestAttachEncryption_WithKeyID(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	evt := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	ct, err := enc.Encrypt([]byte(`{"data":"hello"}`))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	attached, err := AttachEncryption(evt, ct, WithKeyID(KeyID("key-v1")))
	if err != nil {
		t.Fatalf("AttachEncryption: %v", err)
	}

	keyID, err := ExtractKeyID(attached)
	if err != nil {
		t.Fatalf("ExtractKeyID: %v", err)
	}

	if keyID.String() != "key-v1" {
		t.Fatalf("expected key ID %q, got %q", "key-v1", keyID)
	}
}

func TestAttachEncryption_WithAlgorithmAndKeyID(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("NewXChaCha20Poly1305: %v", err)
	}

	evt := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	ct, err := enc.Encrypt([]byte(`{"data":"hello"}`))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	attached, err := AttachEncryption(
		evt, ct,
		func(c *attachConfig) { c.algorithm = XChaCha20Poly1305 },
		WithKeyID(KeyID("key-v2")),
	)
	if err != nil {
		t.Fatalf("AttachEncryption: %v", err)
	}

	alg, _ := ExtractAlgorithm(attached)
	if alg != XChaCha20Poly1305 {
		t.Fatalf("expected %s, got %s", XChaCha20Poly1305, alg)
	}

	keyID, _ := ExtractKeyID(attached)
	if keyID.String() != "key-v2" {
		t.Fatalf("expected key-v2, got %s", keyID)
	}
}

func TestEncryptMiddleware_DetectsAlgorithm(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewAES256GCM(key)

	evt := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	var captured event.Event

	mw := EncryptMiddleware(enc)
	publisher := mw(event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		captured = events[0]

		return nil
	}))

	_ = publisher.Publish(context.Background(), evt)

	alg, err := ExtractAlgorithm(captured)
	if err != nil {
		t.Fatalf("ExtractAlgorithm: %v", err)
	}

	if alg != AES256GCM {
		t.Fatalf("expected algorithm %s, got %s", AES256GCM, alg)
	}
}

func TestEncryptMiddleware_WithKeyID(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewXChaCha20Poly1305(key)

	evt := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	var captured event.Event

	mw := EncryptMiddleware(enc, WithMiddlewareKeyID(KeyID("rotation-key-3")))
	publisher := mw(event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		captured = events[0]

		return nil
	}))

	_ = publisher.Publish(context.Background(), evt)

	keyID, err := ExtractKeyID(captured)
	if err != nil {
		t.Fatalf("ExtractKeyID: %v", err)
	}

	if keyID.String() != "rotation-key-3" {
		t.Fatalf("expected rotation-key-3, got %s", keyID)
	}

	alg, _ := ExtractAlgorithm(captured)
	if alg != XChaCha20Poly1305 {
		t.Fatalf("expected %s, got %s", XChaCha20Poly1305, alg)
	}
}

func TestDecryptMiddleware_RemovesAlgorithmAndKeyID(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, _ := NewAES256GCM(key)

	original := mustEvent(t, "test.created", []byte(`{"data":"hello"}`))

	var decrypted event.Event

	encMw := EncryptMiddleware(enc, WithMiddlewareKeyID(KeyID("k1")))
	decMw := DecryptMiddleware(enc)

	publish := encMw(event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		handler := decMw(func(_ context.Context, evt event.Event) error {
			decrypted = evt

			return nil
		})

		return handler(context.Background(), events[0])
	}))

	_ = publish.Publish(context.Background(), original)

	_, err := ExtractAlgorithm(decrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	keyID, _ := ExtractKeyID(decrypted)
	if keyID != "" {
		t.Fatalf("expected empty key ID after decrypt, got %q", keyID)
	}

	_, err = ExtractCiphertext(decrypted)
	if err == nil {
		t.Fatal("expected ciphertext to be removed after decrypt")
	}
}

func TestExtractAlgorithm_NilEvent(t *testing.T) {
	t.Parallel()

	_, err := ExtractAlgorithm(nil)
	if err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestExtractAlgorithm_NoAlgorithm(t *testing.T) {
	t.Parallel()

	evt := mustEvent(t, "test.created", []byte(`{}`))
	alg, err := ExtractAlgorithm(evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !alg.IsZero() {
		t.Fatalf("expected zero algorithm, got %s", alg)
	}
}

func TestExtractAlgorithm_UnknownAlgorithm(t *testing.T) {
	t.Parallel()

	evt := mustEvent(t, "test.created", []byte(`{}`))

	bad, _ := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithCustom(AlgorithmKey, "unknown-alg"),
	)

	_, err := ExtractAlgorithm(bad)
	if err == nil {
		t.Fatal("expected error for unknown algorithm")
	}
}

func TestDetectAlgorithm(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	aes, _ := NewAES256GCM(key)
	xc, _ := NewXChaCha20Poly1305(key)

	if detectAlgorithm(aes) != AES256GCM {
		t.Fatal("AES encrypter should report AES256GCM algorithm")
	}

	if detectAlgorithm(xc) != XChaCha20Poly1305 {
		t.Fatal("XChaCha20 encrypter should report XChaCha20Poly1305 algorithm")
	}
}

func mustEvent(t *testing.T, eventType string, payload []byte) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		id.NewAggregateID(),
		"Test",
		1,
		payload,
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
}
