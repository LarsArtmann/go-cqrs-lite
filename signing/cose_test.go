package signing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3/internal/testutil"
)

func TestNewCOSEHMAC(t *testing.T) {
	t.Parallel()

	t.Run("valid key", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, signing.MinimumKeyLength)
		_, err := signing.NewCOSEHMAC(key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("short key rejected", func(t *testing.T) {
		t.Parallel()

		key := make([]byte, signing.MinimumKeyLength-1)
		_, err := signing.NewCOSEHMAC(key)
		if err == nil {
			t.Fatal("expected error for short key")
		}
	})
}

func TestNewCOSEEd25519(t *testing.T) {
	t.Parallel()

	pub, priv, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	t.Run("valid signer", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewCOSEEd25519Signer(priv)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid verifier", func(t *testing.T) {
		t.Parallel()

		_, err := signing.NewCOSEEd25519Verifier(pub)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCOSESign1HMAC(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	evt := testutil.MakeTestEvent(t)

	coseBytes, err := signing.SignCOSE1(evt, signer)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if len(coseBytes) == 0 {
		t.Fatal("expected non-empty COSE bytes")
	}

	t.Run("verify with same key", func(t *testing.T) {
		t.Parallel()

		verifier, err := signing.NewCOSEHMAC(key)
		if err != nil {
			t.Fatalf("create verifier: %v", err)
		}

		if err := signing.VerifyCOSE1(evt, verifier, coseBytes); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	t.Run("verify with wrong key", func(t *testing.T) {
		t.Parallel()

		wrongKey := make([]byte, signing.MinimumKeyLength)
		wrongKey[0] = 0xff

		verifier, err := signing.NewCOSEHMAC(wrongKey)
		if err != nil {
			t.Fatalf("create verifier: %v", err)
		}

		if err := signing.VerifyCOSE1(evt, verifier, coseBytes); err == nil {
			t.Fatal("expected verification error for wrong key")
		}
	})

	t.Run("verify with tampered event", func(t *testing.T) {
		t.Parallel()

		tampered := testutil.TamperEvent(t, evt)

		if err := signing.VerifyCOSE1(tampered, signer, coseBytes); err == nil {
			t.Fatal("expected verification error for tampered event")
		}
	})
}

func TestCOSESign1Ed25519(t *testing.T) {
	t.Parallel()

	pub, priv, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	signer, err := signing.NewCOSEEd25519Signer(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	verifier, err := signing.NewCOSEEd25519Verifier(pub)
	if err != nil {
		t.Fatalf("create verifier: %v", err)
	}

	evt := testutil.MakeTestEvent(t)

	coseBytes, err := signing.SignCOSE1(evt, signer)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := signing.VerifyCOSE1(evt, verifier, coseBytes); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestCOSESign1WithKeyID(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	evt := testutil.MakeTestEvent(t)

	coseBytes, err := signing.SignCOSE1(evt, signer, signing.WithCOSEKeyID([]byte("key-1")))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	msg, err := codec.UnmarshalCOSESign1(coseBytes)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	kid, ok := msg.Unprotected[codec.COSEHeaderKid]
	if !ok {
		t.Fatal("expected kid in unprotected header")
	}

	kidBytes, ok := kid.([]byte)
	if !ok {
		t.Fatalf("kid is not a byte string: %T", kid)
	}

	if got, want := string(kidBytes), "key-1"; got != want {
		t.Fatalf("kid = %q, want %q", got, want)
	}
}

func TestCOSESign1WithExternalAAD(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	evt := testutil.MakeTestEvent(t)
	aad := []byte("external-context")

	coseBytes, err := signing.SignCOSE1(evt, signer, signing.WithCOSEExternalAAD(aad))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	t.Run("verify with same AAD", func(t *testing.T) {
		t.Parallel()

		if err := signing.VerifyCOSE1(
			evt,
			signer,
			coseBytes,
			signing.WithCOSEExternalAAD(aad),
		); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	t.Run("verify without AAD fails", func(t *testing.T) {
		t.Parallel()

		if err := signing.VerifyCOSE1(evt, signer, coseBytes); err == nil {
			t.Fatal("expected verification error without AAD")
		}
	})
}

func TestCOSESign1NilInputs(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	t.Run("nil event", func(t *testing.T) {
		t.Parallel()

		if _, err := signing.SignCOSE1(nil, signer); err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("nil signer", func(t *testing.T) {
		t.Parallel()

		evt := testutil.MakeTestEvent(t)

		if _, err := signing.SignCOSE1(evt, nil); err == nil {
			t.Fatal("expected error for nil signer")
		}
	})

	t.Run("nil verifier", func(t *testing.T) {
		t.Parallel()

		evt := testutil.MakeTestEvent(t)

		if err := signing.VerifyCOSE1(evt, nil, nil); err == nil {
			t.Fatal("expected error for nil verifier")
		}
	})
}

func TestCOSESign1AlgorithmMismatch(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	hmacSigner, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	pub, _, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate Ed25519 key pair: %v", err)
	}

	edVerifier, err := signing.NewCOSEEd25519Verifier(pub)
	if err != nil {
		t.Fatalf("create Ed25519 verifier: %v", err)
	}

	evt := testutil.MakeTestEvent(t)

	coseBytes, err := signing.SignCOSE1(evt, hmacSigner)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := signing.VerifyCOSE1(evt, edVerifier, coseBytes); err == nil {
		t.Fatal("expected algorithm mismatch error")
	}
}

func TestCOSEAlgorithmHMAC(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	if got := signer.COSEAlgorithm(); got != codec.COSEAlgHMACSHA256 {
		t.Fatalf("HMAC COSE algorithm = %d, want %d", got, codec.COSEAlgHMACSHA256)
	}
}

func TestCOSEAlgorithmEd25519(t *testing.T) {
	t.Parallel()

	pub, priv, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	signer, err := signing.NewCOSEEd25519Signer(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	verifier, err := signing.NewCOSEEd25519Verifier(pub)
	if err != nil {
		t.Fatalf("create verifier: %v", err)
	}

	if got := signer.COSEAlgorithm(); got != codec.COSEAlgEd25519 {
		t.Fatalf("Ed25519 signer COSE algorithm = %d, want %d", got, codec.COSEAlgEd25519)
	}

	if got := verifier.COSEAlgorithm(); got != codec.COSEAlgEd25519 {
		t.Fatalf("Ed25519 verifier COSE algorithm = %d, want %d", got, codec.COSEAlgEd25519)
	}
}

func TestCOSEHMACSignVerifyRaw(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	data := []byte("hello cose")

	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := signer.Verify(data, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}

	tampered := append([]byte{}, data...)
	tampered[0]++

	if err := signer.Verify(tampered, sig); err == nil {
		t.Fatal("expected verification error for tampered data")
	}
}

func TestCOSEEd25519SignVerifyRaw(t *testing.T) {
	t.Parallel()

	pub, priv, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	signer, err := signing.NewCOSEEd25519Signer(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	verifier, err := signing.NewCOSEEd25519Verifier(pub)
	if err != nil {
		t.Fatalf("create verifier: %v", err)
	}

	data := []byte("hello cose")

	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := verifier.Verify(data, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestCOSESign1RoundTripPreservesEvent(t *testing.T) {
	t.Parallel()

	key := make([]byte, signing.MinimumKeyLength)
	signer, err := signing.NewCOSEHMAC(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	evt := testutil.MakeTestEvent(t)

	coseBytes, err := signing.SignCOSE1(evt, signer)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := signing.VerifyCOSE1(evt, signer, coseBytes); err != nil {
		t.Fatalf("verify: %v", err)
	}

	msg, err := codec.UnmarshalCOSESign1(coseBytes)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(msg.Payload) == 0 {
		t.Fatal("expected non-empty COSE payload")
	}

	protected, err := codec.UnmarshalCOSEProtectedHeader(msg.Protected)
	if err != nil {
		t.Fatalf("unmarshal protected: %v", err)
	}

	alg, ok := protected[codec.COSEHeaderAlg]
	if !ok {
		t.Fatal("expected alg in protected header")
	}

	algID := toInt64(alg)

	if algID != codec.COSEAlgHMACSHA256 {
		t.Fatalf("alg = %v (%T), want %d", alg, alg, codec.COSEAlgHMACSHA256)
	}
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case uint64:
		return int64(x)
	case uint32:
		return int64(x)
	case int32:
		return int64(x)
	default:
		panic("unexpected integer type")
	}
}
