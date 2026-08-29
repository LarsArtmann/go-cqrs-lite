package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// StateTransforms is the plain function pair snapshot.NewTransformedStore
// consumes. It is declared here (structurally identical to any other
// module's shape) so encryption can offer first-class snapshot protection
// without either module importing the other — the consumer wires them:
//
//	codec, err := encryption.SnapshotStateCodec(cipher, "key-2026-08")
//	if err != nil { ... }
//	store, err := snapshot.NewTransformedStore(sqlSnapshots, codec.Protect, codec.Restore)
type StateTransforms struct {
	Protect func(state []byte) ([]byte, error)
	Restore func(state []byte) ([]byte, error)
}

// SnapshotStateCodec returns state transforms that encrypt snapshot state on
// write and decrypt it on read, stamping every envelope with keyID so a later
// key rotation (see [RotatingSnapshotStateCodec]) can tell which snapshots
// need the old key. Routing metadata on the snapshot stays plaintext by
// design — only the state bytes are protected.
func SnapshotStateCodec(cipher EncrypterDecrypter, keyID KeyID) (StateTransforms, error) {
	if cipher == nil {
		return StateTransforms{}, ErrInvalidKey
	}

	return StateTransforms{
		Protect: protectState(cipher, keyID),
		Restore: restoreState(cipher),
	}, nil
}

// RotatingSnapshotStateCodec returns state transforms for a store whose
// active key is activeID/active while older snapshots were written under keys
// resolvable through resolver (e.g. [NewStaticKeyResolver]). Loads consult the
// per-snapshot envelope's key ID: current-key snapshots decrypt with active,
// retired-key snapshots resolve their decrypter and decrypt — which is what
// makes rolling re-encryption (save under the new key on next write) work
// without a migration window.
func RotatingSnapshotStateCodec(
	activeID KeyID,
	active EncrypterDecrypter,
	resolver KeyResolver,
) (StateTransforms, error) {
	if active == nil {
		return StateTransforms{}, ErrInvalidKey
	}

	if resolver == nil {
		return StateTransforms{}, ErrUnknownKeyID
	}

	return StateTransforms{
		Protect: protectState(active, activeID),
		Restore: rotatingRestore(activeID, active, resolver),
	}, nil
}

func protectState(cipher Encrypter, keyID KeyID) func([]byte) ([]byte, error) {
	return func(state []byte) ([]byte, error) {
		if len(state) == 0 {
			return state, nil
		}

		ciphertext, err := cipher.Encrypt(state)
		if err != nil {
			return nil, err
		}

		encoded, err := MarshalEnvelope(
			Envelope{ //nolint:exhaustruct // Version defaults inside MarshalEnvelope
				Ciphertext: ciphertext,
				KeyID:      keyID,
			},
		)
		if err != nil {
			return nil, err
		}

		return []byte(encoded), nil
	}
}

func restoreState(decrypter Decrypter) func([]byte) ([]byte, error) {
	return func(state []byte) ([]byte, error) {
		if len(state) == 0 {
			return state, nil
		}

		env, err := UnmarshalEnvelope(string(state))
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"encryption.snapshot_envelope",
				"snapshot state is not an encryption envelope",
			)
		}

		plaintext, err := decrypter.Decrypt(env.Ciphertext)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"encryption.snapshot_decrypt",
				"decrypt snapshot state",
			)
		}

		return plaintext, nil
	}
}

func rotatingRestore(
	activeID KeyID,
	active Decrypter,
	resolver KeyResolver,
) func([]byte) ([]byte, error) {
	return func(state []byte) ([]byte, error) {
		if len(state) == 0 {
			return state, nil
		}

		env, err := UnmarshalEnvelope(string(state))
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"encryption.snapshot_envelope",
				"snapshot state is not an encryption envelope",
			)
		}

		decrypter := active
		if env.KeyID != "" && env.KeyID != activeID {
			if decrypter, err = resolver.Resolve(env.KeyID); err != nil {
				return nil, err
			}
		}

		plaintext, err := decrypter.Decrypt(env.Ciphertext)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Corruption,
				"encryption.snapshot_decrypt",
				"decrypt snapshot state",
			)
		}

		return plaintext, nil
	}
}
