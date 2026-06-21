package codec

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/fxamacker/cbor/v2"
)

// CBORCodec implements Codec using fxamacker/cbor with canonical encoding
// (RFC 7049: sorted map keys, shortest floats). Canonical mode is deterministic,
// making CBORCodec safe for content-addressed storage and cryptographic signing.
//
// Encoding mode: CanonicalEncOptions (RFC 7049 §3.9 "Canonical CBOR").
// Evaluated CoreDetEncOptions (RFC 7049bis "Core Deterministic") — not adopted
// because it uses SortBytewiseLexical vs SortLengthFirst, changing all output
// bytes and breaking existing stored CBOR data + signatures.
type CBORCodec struct{}

var _ Codec = CBORCodec{}

var (
	encModeOnce sync.Once
	encModeVal  cbor.EncMode
	encModeErr  error
)

func canonicalEncMode() (cbor.EncMode, error) {
	encModeOnce.Do(func() {
		encModeVal, encModeErr = cbor.CanonicalEncOptions().EncMode()
	})

	return encModeVal, encModeErr
}

// CBOREncMode returns the canonical CBOR encoding mode (RFC 7049 sorted map keys).
// Used by storage/pebble and other modules that need deterministic CBOR encoding
// identical to what CBORCodec produces.
func CBOREncMode() (cbor.EncMode, error) {
	return canonicalEncMode()
}

var (
	decModeOnce sync.Once
	decModeVal  cbor.DecMode
	decModeErr  error
)

func canonicalDecMode() (cbor.DecMode, error) {
	decModeOnce.Do(func() {
		//nolint:exhaustruct // only DupMapKey is intentional; all other fields use library defaults
		opts := cbor.DecOptions{DupMapKey: cbor.DupMapKeyEnforcedAPF}
		decModeVal, decModeErr = opts.DecMode()
	})

	return decModeVal, decModeErr
}

// CBORDecMode returns the CBOR decoding mode with duplicate map key enforcement.
// External packages (e.g. storage/pebble) should use this instead of creating
// their own identical DecMode.
func CBORDecMode() (cbor.DecMode, error) {
	return canonicalDecMode()
}

func (CBORCodec) Encoding() Encoding { return EncodingCBOR }

// Encode marshals a value to canonical CBOR bytes with deterministic map ordering.
func (CBORCodec) Encode(v any) ([]byte, error) {
	mode, err := canonicalEncMode()
	if err != nil {
		return nil, fmt.Errorf("codec: CBOR canonical encoding mode: %w", err)
	}

	//nolint:wrapcheck // thin wrapper over cbor EncMode.Marshal
	return mode.Marshal(v)
}

// Decode unmarshals CBOR bytes into a value.
func (CBORCodec) Decode(data []byte, v any) error {
	mode, err := canonicalDecMode()
	if err != nil {
		return fmt.Errorf("codec: CBOR decoding mode: %w", err)
	}

	//nolint:wrapcheck // thin wrapper over cbor DecMode.Unmarshal
	return mode.Unmarshal(data, v)
}

// Diagnose converts CBOR bytes to extended diagnostic notation (EDN) — a
// human-readable representation of CBOR data. Useful for debugging corrupt
// events or inspecting raw CBOR payloads without decoding into a Go struct.
//
//	cborData, _ := codec.CBORCodec{}.Encode(event)
//	diag, _ := codec.Diagnose(cborData)
//	log.Printf("CBOR event: %s", diag)
func Diagnose(data []byte) (string, error) {
	//nolint:wrapcheck // thin wrapper over cbor.Diagnose
	return cbor.Diagnose(data)
}

// EncodeToBuffer writes canonical CBOR encoding of v directly into buf,
// avoiding the allocation that Encode returns. Implements BufferEncoder.
func (CBORCodec) EncodeToBuffer(v any, buf *bytes.Buffer) error {
	mode, err := canonicalEncMode()
	if err != nil {
		return fmt.Errorf("codec: CBOR canonical encoding mode: %w", err)
	}

	//nolint:wrapcheck // thin wrapper over cbor Encoder.Encode
	return mode.NewEncoder(buf).Encode(v)
}
