package event

import (
	"context"
	"fmt"

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
	return streamingIterator(s.inner, s.sourceT, "store", "", func(
		streaming StreamingJournal,
	) (EventIterator, error) {
		return streaming.ReadStream(ctx)
	})
}

// ReadStreamFrom delegates to the inner store's StreamingJournal.ReadStreamFrom
// when supported, applying the source transform per chunk of the returned
// iterator.
func (s *decoratedStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (EventIterator, error) {
	return streamingIterator(s.inner, s.sourceT, "store",
		fmt.Sprintf("limit=%d: ", limit), func(streaming StreamingJournal) (EventIterator, error) {
			return streaming.ReadStreamFrom(ctx, afterEventID, limit)
		})
}

// streamingIterator is the shared StreamingJournal delegation used by both
// decoratedStore and decoratedJournal: it asserts the inner value implements
// StreamingJournal, pulls the iterator via delegate, and applies the source
// transform chunk-wise. A nil sourceT returns the inner iterator unchanged.
// The noun ("store" or "journal") and msgPrefix parameterize the rejection.
func streamingIterator(
	inner any,
	sourceT SourceTransform,
	noun, msgPrefix string,
	delegate func(StreamingJournal) (EventIterator, error),
) (EventIterator, error) {
	streaming, ok := inner.(StreamingJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event."+noun+"_not_streaming",
			msgPrefix+"inner "+noun+" %T does not implement StreamingJournal", inner)
	}

	iter, err := delegate(streaming)
	if err != nil {
		return nil, err
	}

	if sourceT == nil {
		return iter, nil
	}

	return &transformingIterator{inner: iter, sourceT: sourceT}, nil
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
