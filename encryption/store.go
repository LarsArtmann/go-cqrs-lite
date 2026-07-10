package encryption

import (
	"context"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type encryptedStore struct {
	inner event.Store
	enc   Encrypter
	dec   Decrypter
	keyID KeyID
}

var (
	_ event.EventSink       = (*encryptedStore)(nil)
	_ event.EventSource     = (*encryptedStore)(nil)
	_ event.Journal         = (*encryptedStore)(nil)
	_ event.SeekableJournal = (*encryptedStore)(nil)
	_ event.BackwardsSource = (*encryptedStore)(nil)
)

// NewEncryptedStore wraps an event.Store with transparent encryption on
// write and decryption on read. Events are encrypted with the provided
// EncrypterDecrypter before persistence and decrypted on load.
//
// This is a convenience wrapper that composes EncryptMiddleware and
// DecryptMiddleware for consumers who want store-level encryption
// without configuring a bus.
func NewEncryptedStore(
	inner event.Store,
	cipher EncrypterDecrypter,
	opts ...MiddlewareOption,
) (*encryptedStore, error) {
	if inner == nil {
		return nil, ErrNilEvent
	}

	if cipher == nil {
		return nil, ErrInvalidKey
	}

	cfg := middlewareConfig{} //nolint:exhaustruct // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	return &encryptedStore{
		inner: inner,
		enc:   cipher,
		dec:   cipher,
		keyID: cfg.keyID,
	}, nil
}

func (s *encryptedStore) Save(
	ctx context.Context,
	ref id.AggregateRef,
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
	ref id.AggregateRef,
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
	ref id.AggregateRef,
) ([]event.Event, error) {
	events, err := s.inner.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

func (s *encryptedStore) LoadFromVersion(
	ctx context.Context,
	ref id.AggregateRef,
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
	ref id.AggregateRef,
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
	ref id.AggregateRef,
	timestamp time.Time,
) ([]event.Event, error) {
	events, err := s.inner.LoadToTimestamp(ctx, ref, timestamp)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

// ReadAll delegates to the inner store's Journal.ReadAll if supported,
// then decrypts all events.
func (s *encryptedStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	journal, ok := s.inner.(event.Journal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotJournal, errorfamily.Rejection,
			"encryption.store_not_journal", "inner store %T does not implement Journal", s.inner)
	}

	events, err := journal.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

// ReadFrom delegates to the inner store's SeekableJournal.ReadFrom if supported,
// then decrypts all events.
func (s *encryptedStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	seekable, ok := s.inner.(event.SeekableJournal)
	if !ok {
		return nil, errorfamily.Wrapf(
			ErrInnerStoreNotSeekable,
			errorfamily.Rejection,
			"encryption.store_not_seekable",
			"limit=%d: inner store %T does not implement SeekableJournal",
			limit,
			s.inner,
		)
	}

	events, err := seekable.ReadFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"encryption.read_from",
			"limit=%d",
			limit,
		)
	}

	return s.decryptEvents(events)
}

// LoadBackwards delegates to the inner store's BackwardsSource.LoadBackwards
// if supported, then decrypts all events.
func (s *encryptedStore) LoadBackwards(
	ctx context.Context,
	ref id.AggregateRef,
) ([]event.Event, error) {
	backwards, ok := s.inner.(event.BackwardsSource)
	if !ok {
		return nil, errorfamily.Wrapf(
			ErrInnerStoreNotBackwards,
			errorfamily.Rejection,
			"encryption.store_not_backwards",
			"inner store %T does not implement BackwardsSource",
			s.inner,
		)
	}

	events, err := backwards.LoadBackwards(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.decryptEvents(events)
}

// Close closes the underlying store if it implements io.Closer.
func (s *encryptedStore) Close() error {
	if c, ok := s.inner.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

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
