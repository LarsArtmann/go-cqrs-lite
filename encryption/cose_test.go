package encryption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/encryption/v3"
)

func TestNewCOSEXChaCha20Poly1305(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)

	_, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = encryption.NewCOSEXChaCha20Poly1305(make([]byte, 31))
	if err == nil {
		t.Fatal("expected error for wrong key size")
	}
}

func TestNewCOSEAES256GCM(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)

	_, err := encryption.NewCOSEAES256GCM(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = encryption.NewCOSEAES256GCM(make([]byte, 31))
	if err == nil {
		t.Fatal("expected error for wrong key size")
	}
}

func TestCOSEEncryptDecryptXChaCha20(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	plaintext := []byte("hello secret world")

	coseBytes, err := encryption.EncryptCOSE0(plaintext, encrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if len(coseBytes) == 0 {
		t.Fatal("expected non-empty COSE bytes")
	}

	decrypted, err := encryption.DecryptCOSE0(coseBytes, encrypter)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestCOSEEncryptDecryptAESGCM(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEAES256GCM(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	plaintext := []byte("hello secret world")

	coseBytes, err := encryption.EncryptCOSE0(plaintext, encrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := encryption.DecryptCOSE0(coseBytes, encrypter)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("plaintext mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestCOSEEncryptWithExternalAAD(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	plaintext := []byte("hello secret world")
	aad := []byte("additional-data")

	coseBytes, err := encryption.EncryptCOSE0(
		plaintext,
		encrypter,
		encryption.WithCOSEEncryptExternalAAD(aad),
	)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	t.Run("decrypt with same AAD", func(t *testing.T) {
		t.Parallel()

		decrypted, err := encryption.DecryptCOSE0(
			coseBytes,
			encrypter,
			encryption.WithCOSEEncryptExternalAAD(aad),
		)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}

		if string(decrypted) != string(plaintext) {
			t.Fatalf("plaintext mismatch")
		}
	})

	t.Run("decrypt without AAD fails", func(t *testing.T) {
		t.Parallel()

		_, err := encryption.DecryptCOSE0(coseBytes, encrypter)
		if err == nil {
			t.Fatal("expected error decrypting without AAD")
		}
	})

	t.Run("decrypt with wrong AAD fails", func(t *testing.T) {
		t.Parallel()

		_, err := encryption.DecryptCOSE0(
			coseBytes,
			encrypter,
			encryption.WithCOSEEncryptExternalAAD([]byte("wrong")),
		)
		if err == nil {
			t.Fatal("expected error decrypting with wrong AAD")
		}
	})
}

func TestCOSEEncryptAlgorithmMismatch(t *testing.T) {
	t.Parallel()

	xchaKey := make([]byte, 32)
	xchaEncrypter, err := encryption.NewCOSEXChaCha20Poly1305(xchaKey)
	if err != nil {
		t.Fatalf("create XChaCha20 encrypter: %v", err)
	}

	aesKey := make([]byte, 32)
	aesDecrypter, err := encryption.NewCOSEAES256GCM(aesKey)
	if err != nil {
		t.Fatalf("create AES decrypter: %v", err)
	}

	plaintext := []byte("hello secret world")

	coseBytes, err := encryption.EncryptCOSE0(plaintext, xchaEncrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = encryption.DecryptCOSE0(coseBytes, aesDecrypter)
	if err == nil {
		t.Fatal("expected algorithm mismatch error")
	}
}

func TestCOSEEncryptNilInputs(t *testing.T) {
	t.Parallel()

	t.Run("nil encrypter", func(t *testing.T) {
		t.Parallel()

		_, err := encryption.EncryptCOSE0([]byte("hello"), nil)
		if err == nil {
			t.Fatal("expected error for nil encrypter")
		}
	})

	t.Run("nil decrypter", func(t *testing.T) {
		t.Parallel()

		_, err := encryption.DecryptCOSE0([]byte("hello"), nil)
		if err == nil {
			t.Fatal("expected error for nil decrypter")
		}
	})
}

func TestCOSEEncryptTamperedCiphertext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	plaintext := []byte("hello secret world")

	coseBytes, err := encryption.EncryptCOSE0(plaintext, encrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	coseBytes[len(coseBytes)-1] ^= 0xff

	_, err = encryption.DecryptCOSE0(coseBytes, encrypter)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestCOSEAlgorithmXChaCha20(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	if got := encrypter.COSEAlgorithm(); got != codec.COSEAlgChaCha20Poly1305 {
		t.Fatalf("XChaCha20 algorithm = %d, want %d", got, codec.COSEAlgChaCha20Poly1305)
	}
}

func TestCOSEAlgorithmAESGCM(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEAES256GCM(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	if got := encrypter.COSEAlgorithm(); got != codec.COSEAlgAES256GCM {
		t.Fatalf("AES-256-GCM algorithm = %d, want %d", got, codec.COSEAlgAES256GCM)
	}
}

func TestCOSEEncrypt0RoundTripPreservesIV(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	plaintext := []byte("hello secret world")

	coseBytes, err := encryption.EncryptCOSE0(plaintext, encrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	msg, err := codec.UnmarshalCOSEEncrypt0(coseBytes)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	iv, ok := msg.Unprotected[codec.COSEHeaderIV]
	if !ok {
		t.Fatal("expected IV in unprotected header")
	}

	ivBytes, ok := iv.([]byte)
	if !ok {
		t.Fatalf("IV is not a byte string: %T", iv)
	}

	if len(ivBytes) == 0 {
		t.Fatal("expected non-empty IV")
	}

	protected, err := codec.UnmarshalCOSEProtectedHeader(msg.Protected)
	if err != nil {
		t.Fatalf("unmarshal protected: %v", err)
	}

	alg := toInt64(protected[codec.COSEHeaderAlg])
	if alg != codec.COSEAlgChaCha20Poly1305 {
		t.Fatalf("alg = %v, want %d", alg, codec.COSEAlgChaCha20Poly1305)
	}
}

func TestCOSEEncryptEmptyPlaintext(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	encrypter, err := encryption.NewCOSEXChaCha20Poly1305(key)
	if err != nil {
		t.Fatalf("create encrypter: %v", err)
	}

	coseBytes, err := encryption.EncryptCOSE0([]byte{}, encrypter)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := encryption.DecryptCOSE0(coseBytes, encrypter)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty plaintext, got %q", decrypted)
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
