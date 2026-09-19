package encryption

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	ErrInvalidKey error = errorfamily.NewRejection(
		"encryption.invalid_key",
		"encryption key is empty or invalid",
	)

	ErrKeyNotSet error = errorfamily.NewRejection(
		"encryption.key_not_set",
		"encryption key source is empty or unset",
	)

	ErrDecryptionFailed error = errorfamily.NewRejection(
		"encryption.decryption_failed",
		"decryption failed, ciphertext may be corrupt or wrong key",
	)

	ErrNilCiphertext error = errorfamily.NewRejection(
		"encryption.nil_ciphertext",
		"ciphertext is nil or empty",
	)

	ErrNilEvent error = errorfamily.NewRejection(
		"encryption.nil_event",
		"event is nil",
	)

	ErrNilEncrypter error = errorfamily.NewRejection(
		"encryption.nil_encrypter",
		"COSE encrypter is nil",
	)

	ErrNilDecrypter error = errorfamily.NewRejection(
		"encryption.nil_decrypter",
		"COSE decrypter is nil",
	)

	ErrCOSEAlgorithmMismatch error = errorfamily.NewRejection(
		"encryption.cose_algorithm_mismatch",
		"COSE algorithm does not match decrypter",
	)

	// Deprecated: NewEncryptedStore is now built on event.DecorateStore, which
	// reports unsupported inner-store capabilities via the event package's
	// own sentinels. These aliases remain so existing errors.Is checks keep
	// matching. Removed at v5 (ADR-0126).
	ErrInnerStoreNotJournal error = event.ErrInnerStoreNotJournal

	// Deprecated: See ErrInnerStoreNotJournal. Removed at v5 (ADR-0126).
	ErrInnerStoreNotSeekable error = event.ErrInnerStoreNotSeekable

	// Deprecated: See ErrInnerStoreNotJournal. Removed at v5 (ADR-0126).
	ErrInnerStoreNotBackwards error = event.ErrInnerStoreNotBackwards

	ErrUnknownAlgorithm error = errorfamily.NewRejection(
		"encryption.unknown_algorithm",
		"unknown encryption algorithm",
	)

	ErrUnknownAlgorithmID error = errorfamily.NewRejection(
		"encryption.unknown_algorithm_id",
		"unknown algorithm ID in versioned ciphertext",
	)

	ErrUnknownKeyID error = errorfamily.NewRejection(
		"encryption.unknown_key_id",
		"unknown key ID",
	)
)

// wrapInfraBytes returns (result, nil) when err is nil, otherwise
// (nil, errorfamily.WrapInfrastructure(err, code, msg)). Collapses the repeated
// "if err != nil { return nil, WrapInfrastructure(...) }; return x, nil"
// boilerplate in decrypt/derive functions that return a byte slice.
func wrapInfraBytes(result []byte, err error, code, msg string) ([]byte, error) {
	if err == nil {
		return result, nil
	}

	return nil, errorfamily.WrapInfrastructure(err, code, msg)
}
