package signing_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestHMACSigner_New(t *testing.T) {
	t.Parallel()

	t.Run("valid key", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, signing.MinimumKeyLength)
		_, err := signing.NewHMAC(key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("short key rejected", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, signing.MinimumKeyLength-1)
		_, err := signing.NewHMAC(key)
		if err == nil {
			t.Fatal("expected error for short key")
		}
	})

	t.Run("nil key rejected", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewHMAC(nil)
		if err == nil {
			t.Fatal("expected error for nil key")
		}
	})
}

func TestHMACSigner_SignAndVerify(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	copy(key, []byte("my-secret-key-thirty-two-bytes!!"))

	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	evt := makeTestEvent(t)

	t.Run("sign produces non-empty signature", func(t *testing.T) {
		t.Parallel()

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if sig.IsZero() {
			t.Fatal("expected non-zero signature")
		}
	})

	t.Run("verify valid signature", func(t *testing.T) {
		t.Parallel()

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		err = signer.Verify(evt, sig)
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	t.Run("verify tampered event", func(t *testing.T) {
		t.Parallel()

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		// Tamper with payload by creating a new event
		tampered := tamperEvent(t, evt)

		err = signer.Verify(tampered, sig)
		if err == nil {
			t.Fatal("expected verification to fail for tampered event")
		}
	})

	t.Run("verify wrong signature", func(t *testing.T) {
		t.Parallel()

		wrongSig := signing.Signature(make([]byte, 32))

		err := signer.Verify(evt, wrongSig)
		if err == nil {
			t.Fatal("expected verification to fail for wrong signature")
		}
	})

	t.Run("verify nil signature", func(t *testing.T) {
		t.Parallel()

		err := signer.Verify(evt, nil)
		if err == nil {
			t.Fatal("expected error for nil signature")
		}
	})

	t.Run("sign nil event", func(t *testing.T) {
		t.Parallel()

		_, err := signer.Sign(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})
}

func TestHMACSigner_Deterministic(t *testing.T) {
	t.Parallel()

	key := []byte("my-secret-key-thirty-two-bytes!!")

	signer1, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create signer 1: %v", err)
	}

	signer2, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create signer 2: %v", err)
	}

	evt := makeTestEvent(t)

	sig1, err := signer1.Sign(evt)
	if err != nil {
		t.Fatalf("sign 1: %v", err)
	}

	sig2, err := signer2.Sign(evt)
	if err != nil {
		t.Fatalf("sign 2: %v", err)
	}

	if !bytes.Equal(sig1.Bytes(), sig2.Bytes()) {
		t.Fatal("signatures should be deterministic")
	}
}

func TestHMACSigner_DifferentKeys(t *testing.T) {
	t.Parallel()

	key1 := []byte("key-one-thirty-two-bytes-long!!!")
	key2 := []byte("key-two-thirty-two-bytes-long!!!")

	signer1, _ := signing.NewHMAC(key1)
	signer2, _ := signing.NewHMAC(key2)

	evt := makeTestEvent(t)

	sig1, _ := signer1.Sign(evt)
	sig2, _ := signer2.Sign(evt)

	if bytes.Equal(sig1.Bytes(), sig2.Bytes()) {
		t.Fatal("different keys should produce different signatures")
	}
}

func TestEmptyPayloadEvent(t *testing.T) {
	t.Parallel()

	key := []byte("my-secret-key-thirty-two-bytes!!")
	signer, _ := signing.NewHMAC(key)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, err := event.NewEvent("test.empty", aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	sig, signErr := signer.Sign(evt)
	if signErr != nil {
		t.Fatalf("sign: %v", signErr)
	}

	if sig.IsZero() {
		t.Fatal("empty payload event should still produce non-zero signature")
	}

	if verifyErr := signer.Verify(evt, sig); verifyErr != nil {
		t.Fatalf("verify: %v", verifyErr)
	}
}
