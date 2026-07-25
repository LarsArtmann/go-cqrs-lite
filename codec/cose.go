package codec

import (
	"fmt"
	"math"

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

// NormalizeCOSEAlgorithm converts a CBOR-decoded algorithm value to int64.
// CBOR decodes integers as the narrowest Go type that fits, which means
// small positive values arrive as uint64. This helper normalizes all
// integer variants to int64 so callers can compare against the COSEAlg*
// constants. Returns ErrCOSEAlgorithmOverflow for values that exceed
// int64, and ErrCOSEInvalidAlgorithm for non-integer types.
func NormalizeCOSEAlgorithm(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case uint64:
		if val > math.MaxInt64 {
			return 0, errorfamily.Wrapf(
				ErrCOSEAlgorithmOverflow, errorfamily.Rejection,
				"codec.cose_algorithm_overflow",
				"uint64 value %d overflows int64", val,
			)
		}

		return int64(val), nil
	case uint32:
		return int64(val), nil
	default:
		return 0, errorfamily.Wrapf(
			ErrCOSEInvalidAlgorithm, errorfamily.Rejection,
			"codec.cose_invalid_algorithm",
			"expected integer, got %T", v,
		)
	}
}

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

// COSEAlgHeader returns the CBOR-encoded protected header for a COSE message
// that carries only the algorithm identifier (alg). This is the common case
// for COSE_Encrypt0 and COSE_Sign1 messages where the protected header
// contains no additional parameters.
//
// The returned error is already wrapped as an Infrastructure error so
// callers (encryption, signing) don't need per-module error wrapping.
func COSEAlgHeader(alg int64) ([]byte, error) {
	data, err := MarshalCOSEProtectedHeader(map[int64]any{COSEHeaderAlg: alg})
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"codec.cose_marshal_protected", "marshal COSE protected header")
	}

	return data, nil
}

// PrepareCOSESetup applies COSE options to cfg and returns the protected
// header for the given algorithm. Shared by encryption.EncryptCOSE0 and
// signing.SignCOSE1 to eliminate the duplicated apply-opts + alg-extraction +
// header-build boilerplate that art-dupl flagged across those modules.
//
// The caller passes a pointer to its module-local config (e.g.
// coseEncryptConfig or coseSignConfig); the options are applied in place.
// The O type parameter has a ~func(*Cfg) constraint so module-local defined
// option types (COSEEncryptOption, COSESignOption) satisfy it directly.
func PrepareCOSESetup[Cfg any, O ~func(*Cfg)](cfg *Cfg, opts []O, alg int64) ([]byte, error) {
	for _, o := range opts {
		o(cfg)
	}

	return COSEAlgHeader(alg)
}

// UnmarshalCOSEProtectedHeader deserializes a COSE protected header map from its
// CBOR-encoded form.
func UnmarshalCOSEProtectedHeader(data []byte) (map[int64]any, error) {
	if len(data) == 0 {
		return map[int64]any{}, nil
	}

	return decodeCBORRaw[map[int64]any](data, "codec: unmarshal COSE protected header")
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

