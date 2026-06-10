package encryption

import "github.com/larsartmann/go-cqrs-lite/event/v2"

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
)
