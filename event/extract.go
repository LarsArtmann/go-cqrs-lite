package event

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ExtractCustomBytes retrieves a base64-encoded value from event custom metadata.
// Returns (decoded, found, error) where:
//   - found=false, err=nil: key not present or empty
//   - found=true, err=nil: successfully decoded
//   - found=true, err!=nil: key present but base64 corrupt
//
// The caller MUST nil-check the event before calling.
// This helper is shared by signing and encryption to eliminate duplicated
// metadata-extraction boilerplate.
func ExtractCustomBytes(evt Event, key MetadataKey) ([]byte, bool, error) {
	md := evt.Metadata()

	encoded, ok := md.Custom[key]
	if !ok || encoded == "" {
		return nil, false, nil
	}

	decoded, err := DecodeBase64String(encoded)
	if err != nil {
		return nil, true, errorfamily.WrapInfrastructure(
			err,
			"event.decode_custom_bytes",
			"decode base64 from custom metadata",
		)
	}

	return decoded, true, nil
}

// ExtractCustomBytesChecked is the nil-checked, error-wrapped variant of
// ExtractCustomBytes. Returns (bytes, found, error):
//   - evt == nil → returns (nil, false, nilEvtErr)
//   - ExtractCustomBytes failure → returns (nil, false, WrapInfrastructure(err, wrapCode, wrapMsg))
//   - otherwise → forwards the underlying (decoded, found, nil) triple.
//
// Shared by encryption.ExtractCiphertext and signing.ExtractSignature so the
// nil-check + error-wrap boilerplate lives in one place. Callers still own the
// typed-not-found error (ErrNilCiphertext / ErrNilSignature) since those are
// per-package sentinels.
func ExtractCustomBytesChecked(
	evt Event,
	key MetadataKey,
	nilEvtErr error,
	wrapCode, wrapMsg string,
) ([]byte, bool, error) {
	if evt == nil {
		return nil, false, nilEvtErr
	}

	decoded, found, err := ExtractCustomBytes(evt, key)
	if err != nil {
		return nil, false, errorfamily.WrapInfrastructure(err, wrapCode, wrapMsg)
	}

	return decoded, found, nil
}
