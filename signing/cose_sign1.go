package signing

import (
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// SignCOSE1 creates a COSE_Sign1 message for the event, signing the canonical
// event representation.
//
// The protected header contains the algorithm identifier (alg). The
// unprotected header optionally contains the key identifier (kid). The payload
// is the event's canonical signing bytes. The signature is computed over the
// RFC 9052 Sig_structure.
func SignCOSE1(evt event.Event, signer COSESigner, opts ...COSESignOption) ([]byte, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	if signer == nil {
		return nil, ErrNilSigner
	}

	cfg := coseSignConfig{} //nolint:exhaustruct // zero values are ready

	protected, err := codec.PrepareCOSESetup(&cfg, opts, signer.COSEAlgorithm())
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "signing.cose_setup", "prepare COSE setup")
	}

	payload := canonicalPayload(evt)

	externalAAD := cfg.externalAAD
	if externalAAD == nil {
		externalAAD = []byte{}
	}

	toBeSigned, err := codec.SigStructure(protected, externalAAD, payload)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"signing.cose_build_sig_structure",
			"build COSE Sig_structure",
		)
	}

	sig, err := signer.Sign(toBeSigned)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"signing.cose_sign",
			"sign COSE_Sign1 structure",
		)
	}

	msg := codec.COSESign1{
		Protected:   protected,
		Unprotected: nil,
		Payload:     payload,
		Signature:   sig.Bytes(),
	}

	if len(cfg.kid) > 0 {
		msg.Unprotected = map[int64]any{
			codec.COSEHeaderKid: cfg.kid,
		}
	}

	coseBytes, err := codec.MarshalCOSESign1(msg)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"signing.cose_marshal",
			"marshal COSE_Sign1",
		)
	}

	return coseBytes, nil
}

// VerifyCOSE1 verifies a COSE_Sign1 message against the event.
//
// It checks that the COSE payload matches the event's canonical signing bytes,
// that the algorithm identifier in the protected header matches the verifier, and
// that the signature is valid over the reconstructed Sig_structure.
func VerifyCOSE1(
	evt event.Event,
	verifier COSEVerifier,
	coseBytes []byte,
	opts ...COSESignOption,
) error {
	if evt == nil {
		return ErrNilEvent
	}

	if verifier == nil {
		return ErrNilVerifier
	}

	cfg := coseSignConfig{} //nolint:exhaustruct // zero values are ready
	for _, o := range opts {
		o(&cfg)
	}

	msg, err := codec.UnmarshalCOSESign1(coseBytes)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"signing.cose_unmarshal",
			"unmarshal COSE_Sign1",
		)
	}

	expectedPayload := canonicalPayload(evt)
	if !slices.Equal(msg.Payload, expectedPayload) {
		return errorfamily.Wrapf(
			ErrInvalidSignature, errorfamily.Rejection,
			"signing.cose_payload_mismatch",
			"COSE payload does not match event canonical payload",
		)
	}

	protected, err := codec.UnmarshalCOSEProtectedHeader(msg.Protected)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"signing.cose_unmarshal_protected",
			"unmarshal COSE protected header",
		)
	}

	alg, ok := protected[codec.COSEHeaderAlg]
	if !ok {
		return errorfamily.NewRejection(
			"signing.cose_missing_algorithm",
			"COSE protected header is missing alg",
		)
	}

	algID, err := codec.NormalizeCOSEAlgorithm(alg)
	if err != nil {
		return errorfamily.Wrapf(
			ErrInvalidSignature, errorfamily.Rejection,
			"signing.cose_invalid_algorithm",
			"COSE algorithm is not an integer: %v",
			alg,
		)
	}

	if algID != verifier.COSEAlgorithm() {
		return errorfamily.Wrapf(
			ErrInvalidSignature, errorfamily.Rejection,
			"signing.cose_algorithm_mismatch",
			"COSE algorithm %d does not match verifier algorithm %d",
			algID, verifier.COSEAlgorithm(),
		)
	}

	externalAAD := cfg.externalAAD
	if externalAAD == nil {
		externalAAD = []byte{}
	}

	toBeSigned, err := codec.SigStructure(msg.Protected, externalAAD, msg.Payload)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"signing.cose_rebuild_sig_structure",
			"rebuild COSE Sig_structure",
		)
	}

	if err := verifier.Verify(toBeSigned, Signature(msg.Signature)); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"signing.cose_verify",
			"verify COSE_Sign1 signature",
		)
	}

	return nil
}
