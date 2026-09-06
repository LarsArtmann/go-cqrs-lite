package encryption_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
)

func TestGenerateKey(t *testing.T) {
	t.Parallel()

	key, err := encryption.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	if err := encryption.ValidateKey(key); err != nil {
		t.Fatalf("generated key invalid: %v", err)
	}

	other, err := encryption.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (second): %v", err)
	}

	if string(key) == string(other) {
		t.Fatal("two GenerateKey calls returned identical keys")
	}
}

func TestGenerateKeyBase64_RoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := encryption.GenerateKeyBase64()
	if err != nil {
		t.Fatalf("GenerateKeyBase64: %v", err)
	}

	key, err := encryption.DecodeKeyBase64(encoded)
	if err != nil {
		t.Fatalf("DecodeKeyBase64: %v", err)
	}

	if err := encryption.ValidateKey(key); err != nil {
		t.Fatalf("decoded key invalid: %v", err)
	}
}

func TestValidateKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{"nil key", nil, true},
		{"empty key", []byte{}, true},
		{"too short", make([]byte, 16), true},
		{"too long", make([]byte, 64), true},
		{"valid", make([]byte, encryption.KeySize), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := encryption.ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, encryption.ErrInvalidKey) {
				t.Fatalf("error should wrap ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestDecodeKeyBase64(t *testing.T) {
	t.Parallel()

	valid := make([]byte, encryption.KeySize)
	valid[0] = 0x2a

	tests := []struct {
		name     string
		encoded  string
		wantErr  bool
		errIs    error
		contains string
	}{
		{"valid", encryption.EncodeKeyBase64(valid), false, nil, ""},
		{"tolerates whitespace", encryption.EncodeKeyBase64(valid) + "\n", false, nil, ""},
		{"empty string", "", true, encryption.ErrInvalidKey, "empty"},
		{"not base64", "!!!not-base64!!!", true, encryption.ErrInvalidKey, "base64"},
		{"wrong length", encryption.EncodeKeyBase64(make([]byte, 24)), true, encryption.ErrInvalidKey, "24 bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key, err := encryption.DecodeKeyBase64(tt.encoded)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeKeyBase64() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Fatalf("error should wrap %v, got %v", tt.errIs, err)
				}

				if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
					t.Fatalf("error %q should contain %q", err.Error(), tt.contains)
				}

				return
			}

			if string(key) != string(valid) {
				t.Fatal("decoded key does not round-trip")
			}
		})
	}
}

func TestLoadKeyFromEnv(t *testing.T) {
	valid := make([]byte, encryption.KeySize)
	valid[0] = 0x7f

	t.Run("loads a valid key", func(t *testing.T) {
		t.Setenv("TEST_ENCRYPTION_KEY", encryption.EncodeKeyBase64(valid))

		key, err := encryption.LoadKeyFromEnv("TEST_ENCRYPTION_KEY")
		if err != nil {
			t.Fatalf("LoadKeyFromEnv: %v", err)
		}

		if string(key) != string(valid) {
			t.Fatal("loaded key does not match")
		}
	})

	t.Run("unset variable wraps ErrKeyNotSet", func(t *testing.T) {
		_, err := encryption.LoadKeyFromEnv("TEST_ENCRYPTION_KEY_MISSING")
		if !errors.Is(err, encryption.ErrKeyNotSet) {
			t.Fatalf("error should wrap ErrKeyNotSet, got %v", err)
		}

		if !strings.Contains(err.Error(), "TEST_ENCRYPTION_KEY_MISSING") {
			t.Fatalf("error should name the variable, got %q", err.Error())
		}
	})

	t.Run("malformed value wraps ErrInvalidKey", func(t *testing.T) {
		t.Setenv("TEST_ENCRYPTION_KEY", "short")

		_, err := encryption.LoadKeyFromEnv("TEST_ENCRYPTION_KEY")
		if !errors.Is(err, encryption.ErrInvalidKey) {
			t.Fatalf("error should wrap ErrInvalidKey, got %v", err)
		}
	})
}

func TestLoadKeyFromFile(t *testing.T) {
	t.Parallel()

	valid := make([]byte, encryption.KeySize)
	valid[0] = 0x11

	t.Run("loads key with openssl-style trailing newline", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "key.b64")

		writeErr := os.WriteFile(path, []byte(encryption.EncodeKeyBase64(valid)+"\n"), 0o600)
		if writeErr != nil {
			t.Fatalf("write key file: %v", writeErr)
		}

		key, err := encryption.LoadKeyFromFile(path)
		if err != nil {
			t.Fatalf("LoadKeyFromFile: %v", err)
		}

		if string(key) != string(valid) {
			t.Fatal("loaded key does not match")
		}
	})

	t.Run("missing file is checkable via os.ErrNotExist", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "absent.b64")

		_, err := encryption.LoadKeyFromFile(path)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error should wrap os.ErrNotExist, got %v", err)
		}

		if !strings.Contains(err.Error(), path) {
			t.Fatalf("error should name the path, got %q", err.Error())
		}
	})

	t.Run("malformed content wraps ErrInvalidKey", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "garbage.b64")

		writeErr := os.WriteFile(path, []byte("not a key"), 0o600)
		if writeErr != nil {
			t.Fatalf("write key file: %v", writeErr)
		}

		_, err := encryption.LoadKeyFromFile(path)
		if !errors.Is(err, encryption.ErrInvalidKey) {
			t.Fatalf("error should wrap ErrInvalidKey, got %v", err)
		}
	})
}
