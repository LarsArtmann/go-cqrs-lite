package encryption

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// encryptEvent encrypts a single event's payload. Returns the event unchanged
// if the payload is empty. Shared by both EncryptMiddleware and EncryptedStore.
func encryptEvent(evt event.Event, enc Encrypter, keyID KeyID) (event.Event, error) {
	payload := event.PayloadReadOnly(evt)
	if len(payload) == 0 {
		return evt, nil
	}

	ciphertext, err := enc.Encrypt(payload)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.encrypt_event",
			"encrypt event "+string(evt.Type()),
		)
	}

	var attachOpts []AttachOption

	alg := detectAlgorithm(enc)
	if !alg.IsZero() {
		attachOpts = append(attachOpts, func(c *attachConfig) { c.algorithm = alg })
	}

	if !keyID.IsZero() {
		attachOpts = append(attachOpts, WithKeyID(keyID))
	}

	clone, err := AttachEncryption(evt, ciphertext, attachOpts...)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.attach_ciphertext",
			"attach ciphertext to event "+string(evt.Type()),
		)
	}

	return clone, nil
}

// decryptEvent decrypts a single event. Returns the event unchanged if it
// has no ciphertext metadata. Shared by both DecryptMiddleware and EncryptedStore.
func decryptEvent(evt event.Event, dec Decrypter) (event.Event, error) {
	ct, err := ExtractCiphertext(evt)
	if err != nil {
		return evt, nil //nolint:nilerr // empty payload passthrough is intentional
	}

	plaintext, err := dec.Decrypt(ct)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.decrypt_event",
			"decrypt event "+string(evt.Type()),
		)
	}

	md := evt.Metadata().Clone()
	delete(md.Custom, MetadataKey)
	delete(md.Custom, AlgorithmKey)
	delete(md.Custom, KeyIDKey)

	plainEvt, err := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		plaintext,
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(md),
	)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.rebuild_event",
			"rebuild decrypted event "+string(evt.Type()),
		)
	}

	return plainEvt, nil
}
