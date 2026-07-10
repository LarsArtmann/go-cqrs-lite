package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	ErrInvalidKey = errorfamily.NewRejection(
		"encryption.invalid_key",
		"encryption key is empty or invalid",
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

	ErrCOSEAlgorithmOverflow = errorfamily.NewRejection(
		"encryption.cose_algorithm_overflow",
		"COSE algorithm value overflows int64",
	)

	ErrCOSEInvalidAlgorithm = errorfamily.NewRejection(
		"encryption.cose_invalid_algorithm",
		"COSE algorithm value is not an integer",
	)

	ErrInnerStoreNotJournal = errorfamily.NewRejection(
		"encryption.inner_store_not_journal",
		"inner store does not implement event.Journal",
	)

	ErrInnerStoreNotSeekable = errorfamily.NewRejection(
		"encryption.inner_store_not_seekable",
		"inner store does not implement event.SeekableJournal",
	)

	ErrInnerStoreNotBackwards = errorfamily.NewRejection(
		"encryption.inner_store_not_backwards",
		"inner store does not implement event.BackwardsSource",
	)

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
