package encryption_test

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// A damaged marker must be an error: passing the event on would hand
// still-encrypted bytes to the handler as if they were plaintext.
func TestDecryptMiddlewareCorruptMarkerReturnsError(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		"user.created",
		id.NewStreamID(),
		"User",
		1,
		[]byte(`{"secret":"data"}`),
		event.WithCustom(encryption.MetadataKey, "%%%not-base64%%%"),
	)
	if err != nil {
		t.Fatal(err)
	}

	handler, _ := trackHandler()
	wrapped := encryption.DecryptMiddleware(newTestDecrypter(t))(handler)

	err = wrapped(context.Background(), evt)
	if err == nil {
		t.Fatal("expected an error for a damaged ciphertext marker")
	}

	if errorfamily.Classify(err) != errorfamily.Corruption {
		t.Fatalf("expected Corruption, got %v", errorfamily.Classify(err))
	}
}

func TestDecryptMiddlewarePlainEventPassesThrough(t *testing.T) {
	t.Parallel()

	handler, wasCalled := trackHandler()
	wrapped := encryption.DecryptMiddleware(newTestDecrypter(t))(handler)

	plain := makeTestEvent(t, `{"secret":"data"}`)
	if err := wrapped(context.Background(), plain); err != nil {
		t.Fatalf("unencrypted event must pass through: %v", err)
	}

	if !wasCalled() {
		t.Fatal("handler was not called for an unencrypted event")
	}
}

func newTestDecrypter(t *testing.T) encryption.Decrypter {
	t.Helper()

	dec, err := encryption.NewAES256GCM(generateTestKey(t))
	if err != nil {
		t.Fatal(err)
	}

	return dec
}

func trackHandler() (event.Handler, func() bool) {
	called := false

	return func(context.Context, event.Event) error {
		called = true
		return nil
	}, func() bool { return called }
}
