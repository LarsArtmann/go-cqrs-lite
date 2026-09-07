package tursoengine

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Cipher identifies a page-level AEAD cipher for embedded Turso Database
// encryption. The local engine supports the AEGIS family (recommended,
// fastest on AES hardware) and AES-GCM (NIST-approved, for compliance
// frameworks); ChaCha20-Poly1305 exists only in Turso Cloud, not locally.
type Cipher string

const (
	// CipherAEGIS256 is the default recommendation (256-bit key and nonce).
	CipherAEGIS256 Cipher = "aegis256"
	// CipherAEGIS128L trades to a 128-bit key for faster throughput.
	CipherAEGIS128L Cipher = "aegis128l"
	// CipherAEGIS128X2 is AEGIS-128L with 2-way SIMD parallelism.
	CipherAEGIS128X2 Cipher = "aegis128x2"
	// CipherAEGIS128X4 is AEGIS-128L with 4-way SIMD parallelism.
	CipherAEGIS128X4 Cipher = "aegis128x4"
	// CipherAEGIS256X2 is AEGIS-256 with 2-way SIMD parallelism.
	CipherAEGIS256X2 Cipher = "aegis256x2"
	// CipherAEGIS256X4 is AEGIS-256 with 4-way SIMD parallelism.
	CipherAEGIS256X4 Cipher = "aegis256x4"
	// CipherAES128GCM is NIST-approved AES-GCM with a 128-bit key.
	CipherAES128GCM Cipher = "aes128gcm"
	// CipherAES256GCM is NIST-approved AES-GCM with a 256-bit key; the
	// compliance-safe choice for HIPAA/PCI-DSS audits.
	CipherAES256GCM Cipher = "aes256gcm"
)

// cipherKeyLen returns the required raw key length in bytes and whether the
// cipher is known.
func cipherKeyLen(c Cipher) (int, bool) {
	switch c {
	case CipherAEGIS128L, CipherAEGIS128X2, CipherAEGIS128X4, CipherAES128GCM:
		return 16, true
	case CipherAEGIS256, CipherAEGIS256X2, CipherAEGIS256X4, CipherAES256GCM:
		return 32, true
	default:
		return 0, false
	}
}

// WithEncryption enables experimental page-level encryption for embedded
// Turso databases: every page and the WAL are AEAD-encrypted at rest, and a
// wrong key fails decryption explicitly instead of returning garbage. The
// key is hex-encoded — the Turso Database convention (Turso Cloud BYOK uses
// base64 keys and is a different mechanism). Generate one with
// `openssl rand -hex 32`; store it in a secret manager, never in code.
//
// The DSN gains experimental=encryption plus cipher/key parameters at
// construction, so the key never needs to be written into a caller-owned
// DSN string; redactDSN drops encryption params from error messages.
//
// Remote DSNs are rejected: Turso Cloud BYOK keys ride the connection/sync
// layer per request, which this engine does not manage (operators configure
// that via Turso tooling today).
func WithEncryption(cipher Cipher, hexKey string) Option {
	return func(o *options) {
		o.encryption = &encryptionConfig{cipher: cipher, hexKey: hexKey}
	}
}

type encryptionConfig struct {
	cipher Cipher
	hexKey string
}

// applyEncryption validates the cipher/key pair and rewrites the DSN with
// the driver's encryption parameters. Validation errors quote lengths and
// fixes, never key material.
func applyEncryption(dsn string, cfg *encryptionConfig) (string, error) {
	if isRemoteDSN(dsn) {
		return "", errors.New("encryption applies to embedded databases only; remote Turso Cloud BYOK " +
			"keys are per-connection and not configurable on this engine yet")
	}

	keyLen, known := cipherKeyLen(cfg.cipher)
	if !known {
		return "", fmt.Errorf("unknown encryption cipher %q (valid: aegis256, aegis128l, aegis128x2, aegis128x4, "+
			"aegis256x2, aegis256x4, aes128gcm, aes256gcm)", string(cfg.cipher))
	}

	key, err := hex.DecodeString(cfg.hexKey)
	if err != nil {
		return "", fmt.Errorf("encryption key must be hex-encoded (openssl rand -hex %d): %w", keyLen, err)
	}

	if len(key) != keyLen {
		return "", fmt.Errorf("encryption key is %d bytes, want %d for %s (openssl rand -hex %d)",
			len(key), keyLen, cfg.cipher, keyLen)
	}

	return withEncryptionParams(dsn, cfg.cipher, cfg.hexKey)
}

// withEncryptionParams merges the experimental "encryption" flag into the
// DSN and appends the cipher/key parameters. A DSN that already carries
// encryption parameters is rejected: two key sources must never merge
// silently.
func withEncryptionParams(dsn string, cipher Cipher, hexKey string) (string, error) {
	if _, query, hasQuery := strings.Cut(dsn, "?"); hasQuery {
		for _, part := range strings.Split(query, "&") {
			if key, _, _ := strings.Cut(part, "="); key == "encryption_cipher" || key == "encryption_hexkey" {
				return "", fmt.Errorf("DSN already carries %q; remove it from the DSN or drop "+
					"WithEncryption so exactly one key source remains", key)
			}
		}
	}

	merged := withExperimentalToken(dsn, "encryption")

	return merged + "&encryption_cipher=" + url.QueryEscape(string(cipher)) + "&encryption_hexkey=" + hexKey, nil
}
