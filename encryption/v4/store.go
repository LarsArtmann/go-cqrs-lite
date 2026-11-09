package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// EncryptSinkTransform returns an [event.SinkTransform] that encrypts every
// event's payload before persistence. Events with empty payloads pass
// through unchanged.
//
// A nil encrypter yields a transform that rejects every batch with the
// "encryption.nil_encrypter" rejection — the store-side counterpart to how
// EncryptMiddleware guards against nil.
func EncryptSinkTransform(encrypter Encrypter, opts ...MiddlewareOption) event.SinkTransform {
	if encrypter == nil {
		return rejectSink(
			"encryption.nil_encrypter",
			"EncryptSinkTransform called with nil encrypter",
		)
	}

	cfg := middlewareConfig{} //nolint:exhaustruct_v5 // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	return func(events []event.Event) ([]event.Event, error) {
		encrypted := make([]event.Event, 0, len(events))

		for _, evt := range events {
			enc, err := encryptEvent(evt, encrypter, cfg.keyID)
			if err != nil {
				return nil, err
			}

			encrypted = append(encrypted, enc)
		}

		return encrypted, nil
	}
}

// DecryptSourceTransform returns an [event.SourceTransform] that decrypts
// every event's payload after loading. Events without ciphertext metadata
// pass through unchanged.
//
// A nil decrypter yields a transform that rejects every batch with the
// "encryption.nil_decrypter" rejection.
func DecryptSourceTransform(decrypter Decrypter) event.SourceTransform {
	if decrypter == nil {
		return rejectSink(
			"encryption.nil_decrypter",
			"DecryptSourceTransform called with nil decrypter",
		)
	}

	return func(events []event.Event) ([]event.Event, error) {
		decrypted := make([]event.Event, 0, len(events))

		for _, evt := range events {
			dec, err := decryptEvent(evt, decrypter)
			if err != nil {
				return nil, err
			}

			decrypted = append(decrypted, dec)
		}

		return decrypted, nil
	}
}

// rejectSink is the nil-dependency guard shared by both transforms. It
// returns an unnamed func type, which assigns to both SinkTransform and
// SourceTransform.
func rejectSink(code, msg string) func([]event.Event) ([]event.Event, error) {
	return func([]event.Event) ([]event.Event, error) {
		return nil, errorfamily.NewRejection(code, msg)
	}
}

// NewEncryptedStore wraps an event.Store with transparent encryption on
// write and decryption on read. It composes [EncryptSinkTransform] and
// [DecryptSourceTransform] via [event.DecorateStore], so every read path —
// including Journal, SeekableJournal, BackwardsSource, and MultiSink — is
// covered.
func NewEncryptedStore(
	inner event.Store,
	cipher EncrypterDecrypter,
	opts ...MiddlewareOption,
) (event.Store, error) {
	if inner == nil {
		return nil, ErrNilEvent
	}

	if cipher == nil {
		return nil, ErrInvalidKey
	}

	return event.DecorateStore(
		inner,
		EncryptSinkTransform(cipher, opts...),
		DecryptSourceTransform(cipher),
	), nil
}
