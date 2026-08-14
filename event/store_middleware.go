package event

import (
	"context"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// SinkTransform is applied to events before they are persisted to the inner
// store (write side). Use with [DecorateStore].
type SinkTransform func([]Event) ([]Event, error)

// SourceTransform is applied to events after they are loaded from the inner
// store (read side). It runs on every read path: Load, LoadFromVersion,
// LoadToVersion, LoadToTimestamp, ReadAll, ReadFrom, and LoadBackwards.
// Use with [DecorateStore].
type SourceTransform func([]Event) ([]Event, error)

// DecorateStore wraps store with a write-side and/or read-side transform.
// It is the single, central place that preserves ALL store interfaces:
// Store, Journal, SeekableJournal, BackwardsSource, MultiSink, and io.Closer
// are all forwarded to the inner store, with the transforms applied.
//
// A nil sinkT or sourceT is a pass-through (no allocation, no wrapping).
// When both transforms are nil, store is returned unchanged.
//
// The optional interfaces (Journal, SeekableJournal, BackwardsSource,
// MultiSink) are implemented unconditionally: if the inner store does not
// support one, calling that method returns the corresponding
// ErrInnerStoreNot* rejection instead.
//
// DecorateStore panics when store is nil — this is a composition-time
// programmer error, not a runtime condition. Constructors that want to
// return an error should validate nil before calling DecorateStore.
func DecorateStore(store Store, sinkT SinkTransform, sourceT SourceTransform) Store {
	if store == nil {
		panic("event: DecorateStore called with nil store")
	}

	if sinkT == nil && sourceT == nil {
		return store
	}

	return &decoratedStore{inner: store, sinkT: sinkT, sourceT: sourceT}
}

type decoratedStore struct {
	inner   Store
	sinkT   SinkTransform
	sourceT SourceTransform
}

var (
	_ Store           = (*decoratedStore)(nil)
	_ Journal         = (*decoratedStore)(nil)
	_ SeekableJournal = (*decoratedStore)(nil)
	_ BackwardsSource = (*decoratedStore)(nil)
	_ MultiSink       = (*decoratedStore)(nil)
	_ io.Closer       = (*decoratedStore)(nil)
)

// applySink runs the sink transform (pass-through when nil).
func (s *decoratedStore) applySink(events []Event) ([]Event, error) {
	if s.sinkT == nil {
		return events, nil
	}

	return s.sinkT(events)
}

// applySource runs the source transform on an (events, err) pair
// (pass-through when nil or errored).
func (s *decoratedStore) applySource(events []Event, err error) ([]Event, error) {
	if err != nil {
		return nil, err
	}

	if s.sourceT == nil {
		return events, nil
	}

	return s.sourceT(events)
}

func (s *decoratedStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []Event,
	expectedVersion Version,
) error {
	transformed, err := s.applySink(events)
	if err != nil {
		return err
	}

	return s.inner.Save(ctx, ref, transformed, expectedVersion)
}

func (s *decoratedStore) AppendBatch(
	ctx context.Context,
	ref id.StreamRef,
	events []Event,
) error {
	transformed, err := s.applySink(events)
	if err != nil {
		return err
	}

	return s.inner.AppendBatch(ctx, ref, transformed)
}

// SaveMultiBatch applies the sink transform to every entry, then delegates
// to the inner store's MultiSink when supported.
func (s *decoratedStore) SaveMultiBatch(ctx context.Context, entries []MultiBatchEntry) error {
	multi, ok := s.inner.(MultiSink)
	if !ok {
		return errorfamily.Wrapf(ErrInnerStoreNotMultiSink, errorfamily.Rejection,
			"event.store_not_multi_sink", "inner store %T does not implement MultiSink", s.inner)
	}

	transformed := make([]MultiBatchEntry, len(entries))
	for i, entry := range entries {
		events, err := s.applySink(entry.Events)
		if err != nil {
			return err
		}

		transformed[i] = MultiBatchEntry{Ref: entry.Ref, Events: events}
	}

	return multi.SaveMultiBatch(ctx, transformed)
}

func (s *decoratedStore) Load(
	ctx context.Context,
	ref id.StreamRef,
) ([]Event, error) {
	return s.applySource(s.inner.Load(ctx, ref))
}

func (s *decoratedStore) LoadFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version Version,
) ([]Event, error) {
	return s.applySource(s.inner.LoadFromVersion(ctx, ref, version))
}

func (s *decoratedStore) LoadToVersion(
	ctx context.Context,
	ref id.StreamRef,
	maxVersion Version,
) ([]Event, error) {
	return s.applySource(s.inner.LoadToVersion(ctx, ref, maxVersion))
}

func (s *decoratedStore) LoadToTimestamp(
	ctx context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]Event, error) {
	return s.applySource(s.inner.LoadToTimestamp(ctx, ref, maxTime))
}

// ReadAll delegates to the inner store's Journal.ReadAll when supported.
func (s *decoratedStore) ReadAll(ctx context.Context) ([]Event, error) {
	journal, ok := s.inner.(Journal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotJournal, errorfamily.Rejection,
			"event.store_not_journal", "inner store %T does not implement Journal", s.inner)
	}

	return s.applySource(journal.ReadAll(ctx))
}

// ReadFrom delegates to the inner store's SeekableJournal.ReadFrom when supported.
func (s *decoratedStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]Event, error) {
	seekable, ok := s.inner.(SeekableJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotSeekable, errorfamily.Rejection,
			"event.store_not_seekable",
			"limit=%d: inner store %T does not implement SeekableJournal", limit, s.inner)
	}

	return s.applySource(seekable.ReadFrom(ctx, afterEventID, limit))
}

// LoadBackwards delegates to the inner store's BackwardsSource.LoadBackwards
// when supported.
func (s *decoratedStore) LoadBackwards(
	ctx context.Context,
	ref id.StreamRef,
) ([]Event, error) {
	backwards, ok := s.inner.(BackwardsSource)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotBackwards, errorfamily.Rejection,
			"event.store_not_backwards",
			"inner store %T does not implement BackwardsSource", s.inner)
	}

	return s.applySource(backwards.LoadBackwards(ctx, ref))
}

// Close closes the inner store when it implements io.Closer.
func (s *decoratedStore) Close() error {
	if c, ok := s.inner.(io.Closer); ok {
		return c.Close()
	}

	return nil
}
