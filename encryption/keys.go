package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// GenerateKey returns a cryptographically random [KeySize]-byte key suitable
// for any cipher in this package ([NewAES256GCM], [NewXChaCha20Poly1305]).
// It is the SDK-blessed replacement for hand-rolled `crypto/rand` snippets
// and for telling users to run `openssl rand -base64 32`.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)

	if _, err := rand.Read(key); err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"encryption.generate_key",
			"read %d random bytes",
			KeySize,
		)
	}

	return key, nil
}

// GenerateKeyBase64 returns [GenerateKey] encoded as a base64 standard
// string, ready for an environment variable or config file field.
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}

	return EncodeKeyBase64(key), nil
}

// EncodeKeyBase64 serializes a raw key as a base64 standard string, the
// canonical transport format for keys in env vars and config files.
func EncodeKeyBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// ValidateKey reports whether a raw key is usable by this package's
// ciphers: exactly [KeySize] bytes and non-empty. The returned error wraps
// [ErrInvalidKey] and states the observed and required lengths.
func ValidateKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("%w: key is %d bytes, need exactly %d", ErrInvalidKey, len(key), KeySize)
	}

	return nil
}

// DecodeKeyBase64 decodes a base64-encoded key and validates its length.
// Surrounding whitespace (e.g. a trailing newline from a config file) is
// tolerated. The error wraps [ErrInvalidKey] and names the observed byte
// count, so a malformed key fails with "encryption key must be base64 ...
// got N bytes" instead of a low-level crypto error downstream.
func DecodeKeyBase64(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, fmt.Errorf(
			"%w: key is empty, want base64-encoded %d bytes",
			ErrInvalidKey,
			KeySize,
		)
	}

	key, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: value is not valid base64: %w", ErrInvalidKey, err)
	}

	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	return key, nil
}

// LoadKeyFromEnv reads a base64-encoded key from the named environment
// variable. An unset or empty variable returns an error wrapping
// [ErrKeyNotSet]; a malformed value wraps [ErrInvalidKey].
func LoadKeyFromEnv(name string) ([]byte, error) {
	value := os.Getenv(name)
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf(
			"%w: environment variable %q is empty or unset (generate one with encryption.GenerateKeyBase64)",
			ErrKeyNotSet,
			name,
		)
	}

	return DecodeKeyBase64(value)
}

// LoadKeyFromFile reads a base64-encoded key from a file. Trailing
// newlines — what `openssl rand -base64 32 > keyfile` produces — are
// tolerated. Read failures wrap the os error so [os.ErrNotExist] stays
// checkable; a malformed value wraps [ErrInvalidKey].
func LoadKeyFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file %q: %w", path, err)
	}

	key, err := DecodeKeyBase64(string(data))
	if err != nil {
		return nil, fmt.Errorf("key file %q: %w", path, err)
	}

	return key, nil
}
