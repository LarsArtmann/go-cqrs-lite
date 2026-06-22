package encryption

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// MaxHKDFKeyLen is the maximum output length for HKDF-SHA256
// (255 hash blocks of 32 bytes, per RFC 5869).
const MaxHKDFKeyLen = 255 * sha256.Size

// DeriveKey derives a cryptographic key from a master key using HKDF-SHA256.
//
// This is useful for multi-tenant systems where each tenant needs a unique
// encryption key derived from a single master key. The info parameter provides
// domain separation (e.g. "tenant:acme-corp").
//
// Returns ErrInvalidKey if masterKey is empty. Returns a rejection error if
// length is not between 1 and MaxHKDFKeyLen.
func DeriveKey(masterKey []byte, info string, length int) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, ErrInvalidKey
	}

	if length <= 0 || length > MaxHKDFKeyLen {
		return nil, event.NewRejection(
			"encryption.invalid_key_length",
			"derived key length must be between 1 and MaxHKDFKeyLen",
		)
	}

	reader := hkdf.New(sha256.New, masterKey, nil, []byte(info))
	key := make([]byte, length)

	_, err := io.ReadFull(reader, key)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.derive_key",
			"read derived key",
		)
	}

	return key, nil
}
