package signing_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestMultiSignature(t *testing.T) {
	t.Parallel()

	multiSig := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{
				Actor:     signing.Actor("device"),
				Algorithm: signing.AlgorithmEd25519,
				Sig:       []byte("sig1"),
			},
			{
				Actor:     signing.Actor("server"),
				Algorithm: signing.AlgorithmHMACSHA256,
				Sig:       []byte("sig2"),
			},
		},
	}

	t.Run("count", func(t *testing.T) {
		t.Parallel()
		if got, want := multiSig.Count(), 2; got != want {
			t.Fatalf("Count: got %d, want %d", got, want)
		}
	})

	t.Run("has actor", func(t *testing.T) {
		t.Parallel()
		if !multiSig.HasActor(signing.Actor("device")) {
			t.Fatal("expected HasActor(device) = true")
		}
		if multiSig.HasActor(signing.Actor("gateway")) {
			t.Fatal("expected HasActor(gateway) = false")
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		entry := multiSig.Get(signing.Actor("server"))
		if entry == nil {
			t.Fatal("expected entry for server")
		}
		if entry.Algorithm != signing.AlgorithmHMACSHA256 {
			t.Fatalf("algorithm: got %s, want HMAC-SHA256", entry.Algorithm)
		}
		if multiSig.Get(signing.Actor("gateway")) != nil {
			t.Fatal("expected nil for unknown actor")
		}
	})

	t.Run("actors", func(t *testing.T) {
		t.Parallel()
		actors := multiSig.Actors()
		if len(actors) != 2 {
			t.Fatalf("expected 2 actors, got %d", len(actors))
		}
	})
}

func TestMultiSignatureActors(t *testing.T) {
	t.Parallel()

	multiSig := signing.MultiSignature{
		Entries: []signing.SignatureEntry{
			{Actor: signing.Actor("device"), Algorithm: signing.AlgorithmEd25519, Sig: []byte("a")},
			{Actor: signing.Actor("device"), Algorithm: signing.AlgorithmEd25519, Sig: []byte("b")},
			{
				Actor:     signing.Actor("server"),
				Algorithm: signing.AlgorithmHMACSHA256,
				Sig:       []byte("c"),
			},
		},
	}

	actors := multiSig.Actors()
	if len(actors) != 2 {
		t.Fatalf("expected 2 unique actors, got %d: %v", len(actors), actors)
	}
}

// newDeviceMultiSigner creates a test MultiSigner for the "device" actor using Ed25519.
func TestSignatureEntry_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := signing.SignatureEntry{
		Actor:     "device",
		Algorithm: signing.AlgorithmEd25519,
		Sig:       signing.Signature([]byte("test-sig-bytes")),
		SignedAt:  time.Now().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded signing.SignatureEntry

	unmarshalErr := json.Unmarshal(data, &decoded)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	if decoded.Actor != original.Actor ||
		decoded.Algorithm != original.Algorithm ||
		!decoded.SignedAt.Equal(original.SignedAt) {
		t.Fatalf("JSON roundtrip failed: got %+v, want %+v", decoded, original)
	}

	if !bytes.Equal(decoded.Sig, original.Sig) {
		t.Fatal("signature bytes mismatch after JSON roundtrip")
	}
}

func TestMultiSigner_WithClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	edSigner, _ := signing.NewEd25519(privKey)
	edVerifier, _ := signing.NewEd25519Verifier(pubKey)

	deterministic, err := signing.NewMultiSigner(
		signing.Actor("device"), signing.AlgorithmEd25519,
		edSigner,
		signing.WithVerifier(edVerifier),
		signing.WithClock(func() time.Time { return fixedTime }),
	)
	if err != nil {
		t.Fatalf("create deterministic multi-signer: %v", err)
	}

	evt := makeTestEvent(t)
	clone, _ := deterministic.Sign(evt)

	extracted, _ := signing.ExtractMultiSignature(clone)
	entry := extracted.Get(signing.Actor("device"))
	if entry == nil {
		t.Fatal("expected device entry")
	}

	if !entry.SignedAt.Equal(fixedTime) {
		t.Fatalf("SignedAt: got %v, want %v", entry.SignedAt, fixedTime)
	}
}

func TestNewMultiSigner_Validation(t *testing.T) {
	t.Parallel()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, _ := signing.NewHMAC(key)

	t.Run("rejects empty actor", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner("", signing.AlgorithmHMACSHA256, signer)
		if err == nil {
			t.Fatal("expected error for empty actor")
		}
	})

	t.Run("rejects nil signer", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner(signing.Actor("server"), signing.AlgorithmHMACSHA256, nil)
		if err == nil {
			t.Fatal("expected error for nil signer")
		}
	})

	t.Run("rejects empty algorithm", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner(signing.Actor("server"), "", signer)
		if err == nil {
			t.Fatal("expected error for empty algorithm")
		}
	})

	t.Run("rejects nil verifier for Ed25519", func(t *testing.T) {
		t.Parallel()
		_, privKey, _ := ed25519.GenerateKey(nil)
		edSigner, _ := signing.NewEd25519(privKey)

		_, err := signing.NewMultiSigner(
			signing.Actor("device"),
			signing.AlgorithmEd25519,
			edSigner,
		)
		if err == nil {
			t.Fatal("expected error for nil verifier with Ed25519 signer")
		}
	})

	t.Run("rejects nil clock", func(t *testing.T) {
		t.Parallel()
		_, err := signing.NewMultiSigner(
			signing.Actor("server"), signing.AlgorithmHMACSHA256, signer,
			signing.WithClock(nil),
		)
		if err == nil {
			t.Fatal("expected error for nil clock")
		}
	})
}

func TestMultiSigner_Algorithm(t *testing.T) {
	t.Parallel()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, _ := signing.NewHMAC(key)

	multi, err := signing.NewMultiSigner(
		signing.Actor("server"),
		signing.AlgorithmHMACSHA256,
		signer,
	)
	if err != nil {
		t.Fatalf("create multi-signer: %v", err)
	}

	if multi.Algorithm() != signing.AlgorithmHMACSHA256 {
		t.Fatalf(
			"algorithm mismatch: got %s, want %s",
			multi.Algorithm(),
			signing.AlgorithmHMACSHA256,
		)
	}
}

func TestSignatureEntry_Validate(t *testing.T) {
	t.Parallel()

	valid := signing.SignatureEntry{
		Actor:     signing.Actor("device"),
		Algorithm: signing.AlgorithmEd25519,
		Sig:       signing.Signature([]byte("sig")),
		SignedAt:  time.Now(),
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid entry should pass: %v", err)
	}

	t.Run("rejects empty actor", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Actor = ""
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty actor")
		}
	})

	t.Run("rejects empty algorithm", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Algorithm = ""
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty algorithm")
		}
	})

	t.Run("rejects empty sig", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.Sig = nil
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for empty sig")
		}
	})

	t.Run("rejects zero signedAt", func(t *testing.T) {
		t.Parallel()
		entry := valid
		entry.SignedAt = time.Time{}
		if err := entry.Validate(); err == nil {
			t.Fatal("expected error for zero signedAt")
		}
	})
}
