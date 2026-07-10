package codec

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
	errorfamily "github.com/larsartmann/go-error-family"
)

// COSE structure element counts. They are defined as constants to avoid magic
// numbers in validation and to keep the lint checks quiet.
const (
	coseSign1ElementCount    = 4
	coseEncrypt0ElementCount = 3
)

// COSE header parameter labels from the IANA "COSE Header Parameters" registry.
// See https://www.iana.org/assignments/cose/cose.xhtml
const (
	COSEHeaderAlg         int64 = 1
	COSEHeaderCrit        int64 = 2
	COSEHeaderContentType int64 = 3
	COSEHeaderKid         int64 = 4
	COSEHeaderIV          int64 = 5
	COSEHeaderPartialIV   int64 = 6
)

// COSE algorithm identifiers from the IANA "COSE Algorithms" registry.
// These are the algorithms used by the signing and encryption modules.
const (
	COSEAlgHMACSHA256_64    int64 = 4
	COSEAlgHMACSHA256       int64 = 5
	COSEAlgAES256GCM        int64 = 3
	COSEAlgChaCha20Poly1305 int64 = 24
	COSEAlgEdDSA            int64 = -8
	COSEAlgEd25519          int64 = -19
)

// COSESign1 represents a COSE_Sign1 structure as defined in RFC 9052.
// It is a single-signer signed message with protected headers, unprotected headers,
// payload, and signature.
type COSESign1 struct {
	// Protected contains the serialized protected header map (a CBOR-encoded map
	// wrapped in a byte string). It is stored as raw bytes so that signature
	// verification can use the exact byte representation that was signed.
	Protected []byte

	// Unprotected contains header parameters that are not cryptographically
	// protected, such as the key identifier (kid).
	Unprotected map[int64]any

	// Payload is the signed payload. It may be nil for detached content.
	Payload []byte

	// Signature is the raw cryptographic signature value.
	Signature []byte
}

// COSEEncrypt0 represents a COSE_Encrypt0 structure as defined in RFC 9052.
// It is a single-recipient encrypted message with protected headers, unprotected
// headers, and ciphertext.
type COSEEncrypt0 struct {
	// Protected contains the serialized protected header map (a CBOR-encoded map
	// wrapped in a byte string). It is authenticated as additional data (AAD).
	Protected []byte

	// Unprotected contains header parameters that are not cryptographically
	// protected, such as the initialization vector (IV).
	Unprotected map[int64]any

	// Ciphertext is the encrypted payload. It may be nil for detached content.
	Ciphertext []byte
}

// MarshalCOSEProtectedHeader serializes a COSE protected header map to the CBOR
// bytes that are wrapped in a byte string inside the COSE message. The encoding
// uses the canonical CBOR mode shared with CBORCodec.
func MarshalCOSEProtectedHeader(headers map[int64]any) ([]byte, error) {
	return CBOREncMode().Marshal(headers) //nolint:wrapcheck // thin wrapper over CBOR mode
}

// UnmarshalCOSEProtectedHeader deserializes a COSE protected header map from its
// CBOR-encoded form.
func UnmarshalCOSEProtectedHeader(data []byte) (map[int64]any, error) {
	if len(data) == 0 {
		return map[int64]any{}, nil
	}

	var headers map[int64]any

	if err := CBORDecMode().Unmarshal(data, &headers); err != nil {
		return nil, fmt.Errorf("codec: unmarshal COSE protected header: %w", err)
	}

	return headers, nil
}

// MarshalCOSESign1 encodes a COSE_Sign1 structure to CBOR bytes.
func MarshalCOSESign1(msg COSESign1) ([]byte, error) {
	protected := msg.Protected
	if len(protected) == 0 {
		protected = []byte{}
	}

	arr := []any{
		protected,
		msg.Unprotected,
		msg.Payload,
		msg.Signature,
	}

	return CBOREncMode().Marshal(arr) //nolint:wrapcheck // thin wrapper over CBOR mode
}

// UnmarshalCOSESign1 decodes a COSE_Sign1 structure from CBOR bytes.
func UnmarshalCOSESign1(data []byte) (COSESign1, error) {
	var raw []cbor.RawMessage

	if err := CBORDecMode().Unmarshal(data, &raw); err != nil {
		return COSESign1{}, fmt.Errorf("codec: unmarshal COSE_Sign1: %w", err)
	}

	if len(raw) != coseSign1ElementCount {
		return COSESign1{}, errorfamily.Wrapf(
			ErrInvalidCOSESign1, errorfamily.Rejection,
			"codec.cose_sign1_element_count",
			"COSE_Sign1 has %d elements, want %d",
			len(raw), coseSign1ElementCount,
		)
	}

	protected, err := decodeBstr(raw[0])
	if err != nil {
		return COSESign1{}, fmt.Errorf("codec: COSE_Sign1 protected: %w", err)
	}

	unprotected, err := decodeIntMap(raw[1])
	if err != nil {
		return COSESign1{}, fmt.Errorf("codec: COSE_Sign1 unprotected: %w", err)
	}

	payload, err := decodeOptionalBstr(raw[2])
	if err != nil {
		return COSESign1{}, fmt.Errorf("codec: COSE_Sign1 payload: %w", err)
	}

	signature, err := decodeBstr(raw[3])
	if err != nil {
		return COSESign1{}, fmt.Errorf("codec: COSE_Sign1 signature: %w", err)
	}

	return COSESign1{
		Protected:   protected,
		Unprotected: unprotected,
		Payload:     payload,
		Signature:   signature,
	}, nil
}

// MarshalCOSEEncrypt0 encodes a COSE_Encrypt0 structure to CBOR bytes.
func MarshalCOSEEncrypt0(msg COSEEncrypt0) ([]byte, error) {
	protected := msg.Protected
	if len(protected) == 0 {
		protected = []byte{}
	}

	arr := []any{
		protected,
		msg.Unprotected,
		msg.Ciphertext,
	}

	return CBOREncMode().Marshal(arr) //nolint:wrapcheck // thin wrapper over CBOR mode
}

// UnmarshalCOSEEncrypt0 decodes a COSE_Encrypt0 structure from CBOR bytes.
func UnmarshalCOSEEncrypt0(data []byte) (COSEEncrypt0, error) {
	var raw []cbor.RawMessage

	if err := CBORDecMode().Unmarshal(data, &raw); err != nil {
		return COSEEncrypt0{}, fmt.Errorf("codec: unmarshal COSE_Encrypt0: %w", err)
	}

	if len(raw) != coseEncrypt0ElementCount {
		return COSEEncrypt0{}, errorfamily.Wrapf(
			ErrInvalidCOSEEncrypt0, errorfamily.Rejection,
			"codec.cose_encrypt0_element_count",
			"COSE_Encrypt0 has %d elements, want %d",
			len(raw), coseEncrypt0ElementCount,
		)
	}

	protected, err := decodeBstr(raw[0])
	if err != nil {
		return COSEEncrypt0{}, fmt.Errorf("codec: COSE_Encrypt0 protected: %w", err)
	}

	unprotected, err := decodeIntMap(raw[1])
	if err != nil {
		return COSEEncrypt0{}, fmt.Errorf("codec: COSE_Encrypt0 unprotected: %w", err)
	}

	ciphertext, err := decodeOptionalBstr(raw[2])
	if err != nil {
		return COSEEncrypt0{}, fmt.Errorf("codec: COSE_Encrypt0 ciphertext: %w", err)
	}

	return COSEEncrypt0{
		Protected:   protected,
		Unprotected: unprotected,
		Ciphertext:  ciphertext,
	}, nil
}

// SigStructure builds the Sig_structure used for COSE_Sign1 signing and
// verification as defined in RFC 9052 Section 4.4.
func SigStructure(bodyProtected, externalAAD, payload []byte) ([]byte, error) {
	arr := []any{
		"Signature1",
		bodyProtected,
		externalAAD,
		payload,
	}

	return CBOREncMode().Marshal(arr) //nolint:wrapcheck // thin wrapper over CBOR mode
}

// EncStructure0 builds the Enc_structure used for COSE_Encrypt0 encryption and
// decryption as defined in RFC 9052 Section 5.3.
func EncStructure0(bodyProtected, externalAAD []byte) ([]byte, error) {
	arr := []any{
		"Encrypt0",
		bodyProtected,
		externalAAD,
	}

	return CBOREncMode().Marshal(arr) //nolint:wrapcheck // thin wrapper over CBOR mode
}

// decodeBstr decodes a CBOR byte string, accepting nil as an empty byte string.
func decodeBstr(r cbor.RawMessage) ([]byte, error) {
	if isNil(r) {
		return []byte{}, nil
	}

	var out []byte

	if err := CBORDecMode().Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("decode bstr: %w", err)
	}

	return out, nil
}

// decodeOptionalBstr decodes a CBOR byte string or nil value into a byte slice.
// A nil CBOR value returns nil to represent an absent optional field (e.g.,
// detached payload or detached ciphertext).
func decodeOptionalBstr(r cbor.RawMessage) ([]byte, error) {
	if isNil(r) {
		return nil, nil
	}

	var out []byte

	if err := CBORDecMode().Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("decode optional bstr: %w", err)
	}

	return out, nil
}

// decodeIntMap decodes a CBOR map with integer keys.
func decodeIntMap(r cbor.RawMessage) (map[int64]any, error) {
	if isNil(r) {
		return nil, nil //nolint:nilnil // nil represents absent optional header map
	}

	var out map[int64]any

	if err := CBORDecMode().Unmarshal(r, &out); err != nil {
		return nil, fmt.Errorf("decode int map: %w", err)
	}

	return out, nil
}

// isNil reports whether r is a CBOR nil value.
func isNil(r cbor.RawMessage) bool {
	return len(r) == 1 && r[0] == 0xf6
}

// COSESign1String returns a human-readable diagnostic notation of a COSE_Sign1
// message for debugging. It panics only if CBOR diagnosis itself fails, which
// cannot happen for valid CBOR data.
func COSESign1String(data []byte) string {
	diag, err := cbor.Diagnose(data)
	if err != nil {
		return fmt.Sprintf("<diagnose failed: %v>", err)
	}

	return diag
}

// COSEEncrypt0String returns a human-readable diagnostic notation of a
// COSE_Encrypt0 message for debugging.
func COSEEncrypt0String(data []byte) string {
	diag, err := cbor.Diagnose(data)
	if err != nil {
		return fmt.Sprintf("<diagnose failed: %v>", err)
	}

	return diag
}
