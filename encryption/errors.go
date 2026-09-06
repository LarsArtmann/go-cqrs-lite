package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

var (
	ErrInvalidKey = errorfamily.NewRejection(
		"encryption.invalid_key",
		"encryption key is empty or invalid",
	)

	ErrKeyNotSet = errorfamily.NewRejection(
		"encryption.key_not_set",
		"encryption key source is empty or unset",
	)

	ErrDecryptionFailed = errorfamily.NewRejection(
		"encryption.decryption_failed",
		"decryption failed, ciphertext may be corrupt or wrong key",
	)

	ErrNilCiphertext = errorfamily.NewRejection(
		"encryption.nil_ciphertext",
		"ciphertext is nil or empty",
	)

	ErrNilEvent = errorfamily.NewRejection(
		"encryption.nil_event",
		"event is nil",
	)

	ErrNilEncrypter = errorfamily.NewRejection(
		"encryption.nil_encrypter",
		"COSE encrypter is nil",
	)

	ErrNilDecrypter = errorfamily.NewRejection(
		"encryption.nil_decrypter",
		"COSE decrypter is nil",
	)

	ErrCOSEAlgorithmMismatch = errorfamily.NewRejection(
		"encryption.cose_algorithm_mismatch",
		"COSE algorithm does not match decrypter",
	)

	// Deprecated: NewEncryptedStore is now built on event.DecorateStore, which
	// reports unsupported inner-store capabilities via the event package's
	// own sentinels. These aliases remain so existing errors.Is checks keep
	// matching. Removed at v5 (ADR-0126).
	ErrInnerStoreNotJournal = event.ErrInnerStoreNotJournal

	// Deprecated: See ErrInnerStoreNotJournal. Removed at v5 (ADR-0126).
	ErrInnerStoreNotSeekable = event.ErrInnerStoreNotSeekable

	// Deprecated: See ErrInnerStoreNotJournal. Removed at v5 (ADR-0126).
	ErrInnerStoreNotBackwards = event.ErrInnerStoreNotBackwards

	ErrUnknownAlgorithm = errorfamily.NewRejection(
		"encryption.unknown_algorithm",
		"unknown encryption algorithm",
	)

	ErrUnknownAlgorithmID = errorfamily.NewRejection(
		"encryption.unknown_algorithm_id",
		"unknown algorithm ID in versioned ciphertext",
	)

	ErrUnknownKeyID = errorfamily.NewRejection(
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
