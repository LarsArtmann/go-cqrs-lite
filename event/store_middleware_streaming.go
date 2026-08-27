package event

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// LoadStream delegates to the inner store's StreamingSource.LoadStream when
// supported, applying the source transform per chunk of the returned iterator.
func (s *decoratedStore) LoadStream(
	ctx context.Context,
	ref id.StreamRef,
) (EventIterator, error) {
	streaming, ok := s.inner.(StreamingSource)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.store_not_streaming",
			"inner store %T does not implement StreamingSource", s.inner)
	}

	iter, err := streaming.LoadStream(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.wrapIterator(iter), nil
}

// LoadStreamFromVersion delegates to the inner store's
// StreamingSource.LoadStreamFromVersion when supported, applying the source
// transform per chunk of the returned iterator.
func (s *decoratedStore) LoadStreamFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version Version,
) (EventIterator, error) {
	streaming, ok := s.inner.(StreamingSource)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.store_not_streaming",
			"inner store %T does not implement StreamingSource", s.inner)
	}

	iter, err := streaming.LoadStreamFromVersion(ctx, ref, version)
	if err != nil {
		return nil, err
	}

	return s.wrapIterator(iter), nil
}

// ReadStream delegates to the inner store's StreamingJournal.ReadStream when
// supported, applying the source transform per chunk of the returned iterator.
func (s *decoratedStore) ReadStream(ctx context.Context) (EventIterator, error) {
	streaming, ok := s.inner.(StreamingJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.store_not_streaming",
			"inner store %T does not implement StreamingJournal", s.inner)
	}

	iter, err := streaming.ReadStream(ctx)
	if err != nil {
		return nil, err
	}

	return s.wrapIterator(iter), nil
}

// ReadStreamFrom delegates to the inner store's StreamingJournal.ReadStreamFrom
// when supported, applying the source transform per chunk of the returned
// iterator.
func (s *decoratedStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (EventIterator, error) {
	streaming, ok := s.inner.(StreamingJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.store_not_streaming",
			"limit=%d: inner store %T does not implement StreamingJournal", limit, s.inner)
	}

	iter, err := streaming.ReadStreamFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, err
	}

	return s.wrapIterator(iter), nil
}

// wrapIterator applies the source transform chunk-wise. A nil sourceT
// (sink-only decoration) returns the inner iterator unchanged: there is
// nothing to transform on the read side.
func (s *decoratedStore) wrapIterator(iter EventIterator) EventIterator {
	if s.sourceT == nil {
		return iter
	}

	return &transformingIterator{inner: iter, sourceT: s.sourceT}
}
