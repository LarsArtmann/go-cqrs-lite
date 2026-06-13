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
func NewEncryptedStore(inner event.Store, ed EncrypterDecrypter, opts ...MiddlewareOption) (*encryptedStore, error) {
	if inner == nil {
		return nil, ErrNilEvent
	}

	if ed == nil {
		return nil, ErrInvalidKey
	}

	cfg := middlewareConfig{} //nolint:exhaustruct // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	return &encryptedStore{
		inner: inner,
		enc:   ed,
		dec:   ed,
		keyID: cfg.keyID,
	}, nil
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
		encrypted, err := encryptEvent(evt, s.enc, s.keyID)
		if err != nil {
			return nil, err
		}

		result = append(result, encrypted)
	}

	return result, nil
}

func (s *encryptedStore) decryptEvents(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		decrypted, err := decryptEvent(evt, s.dec)
		if err != nil {
			return nil, err
		}

		result = append(result, decrypted)
	}

	return result, nil
}
