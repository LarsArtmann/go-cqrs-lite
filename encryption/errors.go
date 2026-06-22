package encryption

import "github.com/larsartmann/go-cqrs-lite/event/v3"

var (
	ErrInvalidKey = event.NewRejection(
		"encryption.invalid_key",
		"encryption key is empty or invalid",
	)

	ErrDecryptionFailed = event.NewRejection(
		"encryption.decryption_failed",
		"decryption failed, ciphertext may be corrupt or wrong key",
	)

	ErrNilCiphertext = event.NewRejection(
		"encryption.nil_ciphertext",
		"ciphertext is nil or empty",
	)

	ErrNilEvent = event.NewRejection(
		"encryption.nil_event",
		"event is nil",
	)

	ErrInnerStoreNotJournal = event.NewRejection(
		"encryption.inner_store_not_journal",
		"inner store does not implement event.Journal",
	)

	ErrInnerStoreNotSeekable = event.NewRejection(
		"encryption.inner_store_not_seekable",
		"inner store does not implement event.SeekableJournal",
	)

	ErrInnerStoreNotBackwards = event.NewRejection(
		"encryption.inner_store_not_backwards",
		"inner store does not implement event.BackwardsSource",
	)

	ErrUnknownAlgorithm = event.NewRejection(
		"encryption.unknown_algorithm",
		"unknown encryption algorithm",
	)

	ErrUnknownAlgorithmID = event.NewRejection(
		"encryption.unknown_algorithm_id",
		"unknown algorithm ID in versioned ciphertext",
	)

	ErrUnknownKeyID = event.NewRejection(
		"encryption.unknown_key_id",
		"unknown key ID",
	)
)
