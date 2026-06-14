package encryption

import (
	"fmt"
	"strconv"
)

// CiphertextVersion identifies the envelope format version.
type CiphertextVersion byte

const (
	// CiphertextV1 is the first versioned envelope format.
	// Layout: [version:1][algorithm:1][ciphertext:N].
	CiphertextV1 CiphertextVersion = 0x01
)

const (
	algIDAES     byte = 0x01
	algIDXChaCha byte = 0x02
)

// algorithmID maps Algorithm constants to single-byte identifiers for
// compact storage inside the versioned envelope.
//
//nolint:gochecknoglobals // immutable lookup table for envelope serialization
var algorithmID = map[Algorithm]byte{
	AES256GCM:         algIDAES,
	XChaCha20Poly1305: algIDXChaCha,
}

// idToAlgorithm is the reverse lookup for decryption.
//
//nolint:gochecknoglobals // immutable lookup table for envelope deserialization
var idToAlgorithm = map[byte]Algorithm{
	algIDAES:     AES256GCM,
	algIDXChaCha: XChaCha20Poly1305,
}

const versionHeaderLen = 2 // version byte + algorithm byte

// WrapCiphertext prefixes raw ciphertext with a version and algorithm
// identifier, producing a self-describing envelope that does not depend
// on external metadata for decryption.
//
// Layout: [version:1][algorithm:1][ciphertext:N].
//
// Use UnwrapCiphertext to reverse this operation.
func WrapCiphertext(raw Ciphertext, alg Algorithm) (Ciphertext, error) {
	algID, ok := algorithmID[alg]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownAlgorithm, alg)
	}

	result := make(Ciphertext, 0, versionHeaderLen+len(raw))
	result = append(result, byte(CiphertextV1), algID)
	result = append(result, raw...)

	return result, nil
}

// UnwrapCiphertext extracts the algorithm and raw ciphertext from a
// versioned envelope. Returns the algorithm, raw ciphertext, and error.
//
// If the input does not start with a recognized version byte, it is
// treated as raw (unversioned) ciphertext with an empty Algorithm,
// preserving backward compatibility with ciphertexts produced before
// versioning was introduced.
func UnwrapCiphertext(data Ciphertext) (Algorithm, Ciphertext, error) {
	if len(data) < versionHeaderLen {
		return "", data, nil
	}

	version := CiphertextVersion(data[0])
	if version != CiphertextV1 {
		return "", data, nil
	}

	algID := data[1]

	alg, ok := idToAlgorithm[algID]
	if !ok {
		return "", nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithmID,
			strconv.FormatUint(uint64(algID), hexBase))
	}

	return alg, data[versionHeaderLen:], nil
}

const hexBase = 16
