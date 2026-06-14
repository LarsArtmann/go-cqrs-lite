package encryption

import "fmt"

// CiphertextVersion identifies the envelope format version.
type CiphertextVersion byte

const (
	// CiphertextV1 is the first versioned envelope format.
	// Layout: [version:1][algorithm:1][ciphertext:N]
	CiphertextV1 CiphertextVersion = 0x01
)

// algorithmID maps Algorithm constants to single-byte identifiers for
// compact storage inside the versioned envelope.
var algorithmID = map[Algorithm]byte{
	AES256GCM:         0x01,
	XChaCha20Poly1305: 0x02,
}

var idToAlgorithm = map[byte]Algorithm{
	0x01: AES256GCM,
	0x02: XChaCha20Poly1305,
}

const versionHeaderLen = 2 // version byte + algorithm byte

// WrapCiphertext prefixes raw ciphertext with a version and algorithm
// identifier, producing a self-describing envelope that does not depend
// on external metadata for decryption.
//
// Layout: [version:1][algorithm:1][ciphertext:N]
//
// Use UnwrapCiphertext to reverse this operation.
func WrapCiphertext(raw Ciphertext, alg Algorithm) (Ciphertext, error) {
	algID, ok := algorithmID[alg]
	if !ok {
		return nil, fmt.Errorf("encryption.wrap_ciphertext: unknown algorithm %q", alg)
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
		// Too short for a version header; treat as raw unversioned ciphertext.
		return "", data, nil
	}

	version := CiphertextVersion(data[0])
	if version != CiphertextV1 {
		// Not a recognized version; treat as raw unversioned ciphertext.
		return "", data, nil
	}

	algID := data[1]
	alg, ok := idToAlgorithm[algID]
	if !ok {
		return "", nil, fmt.Errorf(
			"encryption.unwrap_ciphertext: unknown algorithm ID 0x%02x",
			algID,
		)
	}

	return alg, data[versionHeaderLen:], nil
}
