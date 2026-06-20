package encryption_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func generateTestKey(t *testing.T) []byte {
	t.Helper()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	return key
}

func makeTestEvent(t *testing.T, payload string) event.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("user.created", aggID, "User", 1, []byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	return evt
}

func TestAttachExtractEncryption(t *testing.T) {
	t.Parallel()

	key := generateTestKey(t)
	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		t.Fatal(err)
	}

	evt := makeTestEvent(t, `{"secret":"data"}`)

	ct, err := enc.Encrypt(evt.Payload())
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := encryption.AttachEncryption(evt, ct)
	if err != nil {
		t.Fatal(err)
	}

	if !encryption.HasEncryption(encrypted) {
		t.Error("encrypted event should have encryption marker")
	}

	extracted, err := encryption.ExtractCiphertext(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := enc.Decrypt(extracted)
	if err != nil {
		t.Fatal(err)
	}

	if string(plaintext) != `{"secret":"data"}` {
		t.Errorf("plaintext = %q, want original payload", plaintext)
	}
}

func TestExtractCiphertext_PlainEvent(t *testing.T) {
	t.Parallel()

	evt := makeTestEvent(t, `{"public":"data"}`)

	if encryption.HasEncryption(evt) {
		t.Error("plain event should not have encryption marker")
	}

	_, err := encryption.ExtractCiphertext(evt)
	if err == nil {
		t.Error("expected error extracting from unencrypted event")
	}
}

func TestEncryptMiddleware(t *testing.T) {
	t.Parallel()

	key := generateTestKey(t)
	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		t.Fatal(err)
	}

	mw := encryption.EncryptMiddleware(enc)

	evt := makeTestEvent(t, `{"ssn":"123-45-6789"}`)

	var captured event.Event
	inner := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		if len(events) > 0 {
			captured = events[0]
		}

		return nil
	})

	if err := mw(inner).Publish(context.Background(), evt); err != nil {
		t.Fatal(err)
	}

	if !encryption.HasEncryption(captured) {
		t.Error("middleware should add encryption marker")
	}

	ct, err := encryption.ExtractCiphertext(captured)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}

	if string(plaintext) != `{"ssn":"123-45-6789"}` {
		t.Errorf("decrypted payload = %q", plaintext)
	}
}

func TestDecryptMiddleware(t *testing.T) {
	t.Parallel()

	key := generateTestKey(t)
	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		t.Fatal(err)
	}

	evt := makeTestEvent(t, `{"name":"Bob"}`)

	ct, err := enc.Encrypt(evt.Payload())
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := encryption.AttachEncryption(evt, ct)
	if err != nil {
		t.Fatal(err)
	}

	mw := encryption.DecryptMiddleware(enc)

	var captured event.Event
	handler := func(_ context.Context, e event.Event) error {
		captured = e

		return nil
	}

	if err := mw(handler)(context.Background(), encrypted); err != nil {
		t.Fatal(err)
	}

	if string(captured.Payload()) != `{"name":"Bob"}` {
		t.Errorf("decrypted payload = %q", captured.Payload())
	}
}

func TestEncryptMiddleware_NilEncrypter(t *testing.T) {
	t.Parallel()

	mw := encryption.EncryptMiddleware(nil)

	inner := event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return nil
	})

	err := mw(inner).Publish(context.Background(), makeTestEvent(t, `{}`))
	if err == nil {
		t.Error("expected error with nil encrypter")
	}
}

func TestDecryptMiddleware_NilDecrypter(t *testing.T) {
	t.Parallel()

	mw := encryption.DecryptMiddleware(nil)

	err := mw(func(_ context.Context, _ event.Event) error { return nil })(
		context.Background(), makeTestEvent(t, `{}`),
	)
	if err == nil {
		t.Error("expected error with nil decrypter")
	}
}
