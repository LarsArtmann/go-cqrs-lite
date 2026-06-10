package encryption

import (
	"crypto/rand"
	"encoding/json"
	"testing"
)

func FuzzAES256GCM_Roundtrip(f *testing.F) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewAES256GCM(key)
	if err != nil {
		f.Fatalf("NewAES256GCM: %v", err)
	}

	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Add([]byte{0, 1, 2, 255, 254, 253})
	f.Add(make([]byte, 4096))

	f.Fuzz(func(t *testing.T, plaintext []byte) {
		fuzzRoundtrip(t, enc, plaintext)
	})
}

func fuzzRoundtrip(t *testing.T, enc EncrypterDecrypter, plaintext []byte) {
	t.Helper()

	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if len(plaintext) == 0 {
		if !ct.IsZero() {
			t.Fatal("empty plaintext should produce zero ciphertext")
		}

		return
	}

	decrypted, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func FuzzXChaCha20_Roundtrip(f *testing.F) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewXChaCha20Poly1305(key)
	if err != nil {
		f.Fatalf("NewXChaCha20Poly1305: %v", err)
	}

	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Add([]byte{0, 1, 2, 255, 254, 253})
	f.Add(make([]byte, 4096))

	f.Fuzz(func(t *testing.T, plaintext []byte) {
		fuzzRoundtrip(t, enc, plaintext)
	})
}

func FuzzAES256GCM_CorruptCiphertext(f *testing.F) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewAES256GCM(key)
	if err != nil {
		f.Fatalf("NewAES256GCM: %v", err)
	}

	f.Add([]byte("test data"), byte(1))
	f.Add([]byte("another test"), byte(255))

	f.Fuzz(func(t *testing.T, plaintext []byte, flip byte) {
		if len(plaintext) == 0 {
			return
		}

		fuzzCorruptCiphertext(t, enc, plaintext, flip)
	})
}

func fuzzCorruptCiphertext(t *testing.T, enc EncrypterDecrypter, plaintext []byte, flip byte) {
	t.Helper()

	if len(plaintext) == 0 {
		return
	}

	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	corrupt := ct.Bytes()
	if len(corrupt) == 0 {
		return
	}

	idx := len(corrupt) / 2
	original := corrupt[idx]
	corrupt[idx] ^= flip
	if corrupt[idx] == original {
		corrupt[idx] ^= 0x01
	}

	_, err = enc.Decrypt(Ciphertext(corrupt))
	if err == nil {
		t.Fatal("expected error for corrupt ciphertext")
	}
}

func FuzzXChaCha20_CorruptCiphertext(f *testing.F) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	enc, err := NewXChaCha20Poly1305(key)
	if err != nil {
		f.Fatalf("NewXChaCha20Poly1305: %v", err)
	}

	f.Add([]byte("test data"), byte(1))
	f.Add([]byte("another test"), byte(255))

	f.Fuzz(func(t *testing.T, plaintext []byte, flip byte) {
		fuzzCorruptCiphertext(t, enc, plaintext, flip)
	})
}

func FuzzCiphertext_JSONRoundtrip(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte{0, 1, 2, 255})

	f.Fuzz(func(t *testing.T, data []byte) {
		original := Ciphertext(data)

		encoded, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}

		var decoded Ciphertext
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("UnmarshalJSON: %v", err)
		}

		if !original.Equal(decoded) {
			t.Fatalf("roundtrip failed: got %v, want %v", decoded, original)
		}
	})
}
