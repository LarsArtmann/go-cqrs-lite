package multisig_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/signing/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2/multisig"
)

func newDeviceMultiSigner(t *testing.T) (*multisig.MultiSigner, ed25519.PublicKey) {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, signerErr := signing.NewEd25519(privKey)
	if signerErr != nil {
		t.Fatalf("create signer: %v", signerErr)
	}

	verifier, verifierErr := signing.NewEd25519Verifier(pubKey)
	if verifierErr != nil {
		t.Fatalf("create verifier: %v", verifierErr)
	}

	deviceMulti, err := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmEd25519,
		signer,
		multisig.WithVerifier(verifier),
	)
	if err != nil {
		t.Fatalf("create device multi-signer: %v", err)
	}

	return deviceMulti, pubKey
}

func newServerMultiSigner(t *testing.T) *multisig.MultiSigner {
	t.Helper()

	key := []byte("server-secret-key-thirty-two-by!")
	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	serverMulti, err := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		signer,
	)
	if err != nil {
		t.Fatalf("create server multi-signer: %v", err)
	}

	return serverMulti
}

func TestMultiSignature(t *testing.T) {
	t.Parallel()

	multiSig := multisig.MultiSignature{
		Entries: []multisig.SignatureEntry{
			{
				Actor:     multisig.Actor("device"),
				Algorithm: multisig.AlgorithmEd25519,
				Sig:       []byte("sig1"),
			},
			{
				Actor:     multisig.Actor("server"),
				Algorithm: multisig.AlgorithmHMACSHA256,
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
		if !multiSig.HasActor(multisig.Actor("device")) {
			t.Fatal("expected HasActor(device) = true")
		}
		if multiSig.HasActor(multisig.Actor("gateway")) {
			t.Fatal("expected HasActor(gateway) = false")
		}
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		entry := multiSig.Get(multisig.Actor("server"))
		if entry == nil {
			t.Fatal("expected entry for server")
		}
		if entry.Algorithm != multisig.AlgorithmHMACSHA256 {
			t.Fatalf("algorithm: got %s, want HMAC-SHA256", entry.Algorithm)
		}
		if multiSig.Get(multisig.Actor("gateway")) != nil {
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

	multiSig := multisig.MultiSignature{
		Entries: []multisig.SignatureEntry{
			{
				Actor:     multisig.Actor("device"),
				Algorithm: multisig.AlgorithmEd25519,
				Sig:       []byte("a"),
			},
			{
				Actor:     multisig.Actor("device"),
				Algorithm: multisig.AlgorithmEd25519,
				Sig:       []byte("b"),
			},
			{
				Actor:     multisig.Actor("server"),
				Algorithm: multisig.AlgorithmHMACSHA256,
				Sig:       []byte("c"),
			},
		},
	}

	actors := multiSig.Actors()
	if len(actors) != 2 {
		t.Fatalf("expected 2 unique actors, got %d: %v", len(actors), actors)
	}
}

func TestSignatureEntry_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := multisig.SignatureEntry{
		Actor:     "device",
		Algorithm: multisig.AlgorithmEd25519,
		Sig:       signing.Signature([]byte("test-sig-bytes")),
		SignedAt:  time.Now().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded multisig.SignatureEntry

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

	deterministic, err := multisig.NewMultiSigner(
		multisig.Actor("device"), multisig.AlgorithmEd25519,
		edSigner,
		multisig.WithVerifier(edVerifier),
		multisig.WithClock(func() time.Time { return fixedTime }),
	)
	if err != nil {
		t.Fatalf("create deterministic multi-signer: %v", err)
	}

	evt := makeTestEvent(t)
	clone, _ := deterministic.Sign(evt)

	extracted, _ := multisig.ExtractMultiSignature(clone)
	entry := extracted.Get(multisig.Actor("device"))
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
		_, err := multisig.NewMultiSigner("", multisig.AlgorithmHMACSHA256, signer)
		if err == nil {
			t.Fatal("expected error for empty actor")
		}
	})

	t.Run("rejects nil signer", func(t *testing.T) {
		t.Parallel()
		_, err := multisig.NewMultiSigner(
			multisig.Actor("server"),
			multisig.AlgorithmHMACSHA256,
			nil,
		)
		if err == nil {
			t.Fatal("expected error for nil signer")
		}
	})

	t.Run("rejects empty algorithm", func(t *testing.T) {
		t.Parallel()
		_, err := multisig.NewMultiSigner(multisig.Actor("server"), "", signer)
		if err == nil {
			t.Fatal("expected error for empty algorithm")
		}
	})

	t.Run("rejects nil verifier for Ed25519", func(t *testing.T) {
		t.Parallel()
		_, privKey, _ := ed25519.GenerateKey(nil)
		edSigner, _ := signing.NewEd25519(privKey)

		_, err := multisig.NewMultiSigner(
			multisig.Actor("device"),
			multisig.AlgorithmEd25519,
			edSigner,
		)
		if err == nil {
			t.Fatal("expected error for nil verifier with Ed25519 signer")
		}
	})

	t.Run("rejects nil clock", func(t *testing.T) {
		t.Parallel()
		_, err := multisig.NewMultiSigner(
			multisig.Actor("server"), multisig.AlgorithmHMACSHA256, signer,
			multisig.WithClock(nil),
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

	multi, err := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		signer,
	)
	if err != nil {
		t.Fatalf("create multi-signer: %v", err)
	}

	if multi.Algorithm() != multisig.AlgorithmHMACSHA256 {
		t.Fatalf(
			"algorithm mismatch: got %s, want %s",
			multi.Algorithm(),
			multisig.AlgorithmHMACSHA256,
		)
	}
}

func TestSignatureEntry_Validate(t *testing.T) {
	t.Parallel()

	valid := multisig.SignatureEntry{
		Actor:     multisig.Actor("device"),
		Algorithm: multisig.AlgorithmEd25519,
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
