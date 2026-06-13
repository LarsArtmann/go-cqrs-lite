package encryption

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

type encryptedStore struct {
	inner event.Store
	enc   Encrypter
	dec   Decrypter
	keyID KeyID
}

var (
	_ event.EventSink   = (*encryptedStore)(nil)
	_ event.EventSource = (*encryptedStore)(nil)
)

// NewEncryptedStore wraps an event.Store with transparent encryption on
// write and decryption on read. Events are encrypted with the provided
// EncrypterDecrypter before persistence and decrypted on load.
//
// This is a convenience wrapper that composes EncryptMiddleware and
// DecryptMiddleware for consumers who want store-level encryption
// without configuring a bus.
func NewEncryptedStore(inner event.Store, ed EncrypterDecrypter, opts ...MiddlewareOption) *encryptedStore {
	cfg := middlewareConfig{} //nolint:exhaustruct // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	return &encryptedStore{
		inner: inner,
		enc:   ed,
		dec:   ed,
		keyID: cfg.keyID,
	}
}

func (s *encryptedStore) Save(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	encrypted, err := s.encryptEvents(events)
	if err != nil {
		return err
	}

	return s.inner.Save(ctx, ref, encrypted, expectedVersion)
}

func (s *encryptedStore) AppendBatch(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	encrypted, err := s.encryptEvents(events)
	if err != nil {
		return err
	}

	return s.inner.AppendBatch(ctx, ref, encrypted)
}

func (s *encryptedStore) Load(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	events, err := s.inner.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

func (s *encryptedStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	fromVersion event.Version,
) ([]event.Event, error) {
	events, err := s.inner.LoadFromVersion(ctx, ref, fromVersion)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

func (s *encryptedStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	toVersion event.Version,
) ([]event.Event, error) {
	events, err := s.inner.LoadToVersion(ctx, ref, toVersion)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

func (s *encryptedStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	timestamp time.Time,
) ([]event.Event, error) {
	events, err := s.inner.LoadToTimestamp(ctx, ref, timestamp)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

// Close closes the underlying store.
func (s *encryptedStore) Close() error { return s.inner.Close() }

func (s *encryptedStore) encryptEvents(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		payload := event.PayloadReadOnly(evt)
		if len(payload) == 0 {
			result = append(result, evt)

			continue
		}

		ct, err := s.enc.Encrypt(payload)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"encryption.encrypt_event",
				"encrypt event "+string(evt.Type()),
			)
		}

		var attachOpts []AttachOption

		alg := detectAlgorithm(s.enc)
		if !alg.IsZero() {
			attachOpts = append(attachOpts, func(c *attachConfig) { c.algorithm = alg })
		}

		if !s.keyID.IsZero() {
			attachOpts = append(attachOpts, WithKeyID(s.keyID))
		}

		clone, err := AttachEncryption(evt, ct, attachOpts...)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"encryption.attach_ciphertext",
				"attach ciphertext to event "+string(evt.Type()),
			)
		}

		result = append(result, clone)
	}

	return result, nil
}

func (s *encryptedStore) decryptEvents(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		ct, err := ExtractCiphertext(evt)
		if err != nil {
			result = append(result, evt)

			continue
		}

		plaintext, err := s.dec.Decrypt(ct)
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

		result = append(result, plainEvt)
	}

	return result, nil
}
