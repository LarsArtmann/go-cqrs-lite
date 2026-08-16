package event

import (
	"context"
	"errors"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// journalTransformChunkSize is the number of events a transformingIterator
// pulls from the inner iterator per SourceTransform application. Streaming
// transforms are batch-based ([]Event in, []Event out), so the iterator
// buffers one chunk at a time to keep memory bounded.
const journalTransformChunkSize = 128

// DecorateJournal wraps journal with a read-side transform. It is the journal
// counterpart of [DecorateStore] (ADR-0126): the single, central place that
// preserves ALL journal interfaces — Journal, SeekableJournal,
// StreamingJournal, and io.Closer are all forwarded to the inner journal,
// with the transform applied on every read path.
//
// Journals have no write side, so there is no sink transform.
//
// A nil sourceT is a pass-through: journal is returned unchanged.
//
// The optional interfaces (SeekableJournal, StreamingJournal) are implemented
// unconditionally: if the inner journal does not support one, calling that
// method returns ErrInnerStoreNotSeekable / ErrInnerStoreNotStreaming.
//
// Streaming reads (ReadStream, ReadStreamFrom) apply sourceT per chunk of
// 128 events: the transform sees each chunk independently, the same window a
// paged ReadFrom call would see. Transforms must not reorder events across
// chunk boundaries; use the slice-based reads if global reordering matters.
//
// DecorateJournal panics when journal is nil — this is a composition-time
// programmer error, not a runtime condition. Constructors that want to
// return an error should validate nil before calling DecorateJournal.
func DecorateJournal(journal Journal, sourceT SourceTransform) Journal {
	if journal == nil {
		panic("event: DecorateJournal called with nil journal")
	}

	if sourceT == nil {
		return journal
	}

	return &decoratedJournal{inner: journal, sourceT: sourceT}
}

type decoratedJournal struct {
	inner   Journal
	sourceT SourceTransform
}

var (
	_ Journal          = (*decoratedJournal)(nil)
	_ SeekableJournal  = (*decoratedJournal)(nil)
	_ StreamingJournal = (*decoratedJournal)(nil)
	_ io.Closer        = (*decoratedJournal)(nil)
)

// applySource runs the source transform on an (events, err) pair
// (pass-through when errored).
func (j *decoratedJournal) applySource(events []Event, err error) ([]Event, error) {
	if err != nil {
		return nil, err
	}

	return j.sourceT(events)
}

func (j *decoratedJournal) ReadAll(ctx context.Context) ([]Event, error) {
	return j.applySource(j.inner.ReadAll(ctx))
}

// ReadFrom delegates to the inner journal's SeekableJournal.ReadFrom when
// supported.
func (j *decoratedJournal) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]Event, error) {
	seekable, ok := j.inner.(SeekableJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotSeekable, errorfamily.Rejection,
			"event.journal_not_seekable",
			"limit=%d: inner journal %T does not implement SeekableJournal", limit, j.inner)
	}

	return j.applySource(seekable.ReadFrom(ctx, afterEventID, limit))
}

// ReadStream delegates to the inner journal's StreamingJournal.ReadStream
// when supported, applying the transform per chunk.
func (j *decoratedJournal) ReadStream(ctx context.Context) (EventIterator, error) {
	streaming, ok := j.inner.(StreamingJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.journal_not_streaming",
			"inner journal %T does not implement StreamingJournal", j.inner)
	}

	iter, err := streaming.ReadStream(ctx)
	if err != nil {
		return nil, err
	}

	return &transformingIterator{inner: iter, sourceT: j.sourceT}, nil
}

// ReadStreamFrom delegates to the inner journal's
// StreamingJournal.ReadStreamFrom when supported, applying the transform per
// chunk.
func (j *decoratedJournal) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (EventIterator, error) {
	streaming, ok := j.inner.(StreamingJournal)
	if !ok {
		return nil, errorfamily.Wrapf(ErrInnerStoreNotStreaming, errorfamily.Rejection,
			"event.journal_not_streaming",
			"limit=%d: inner journal %T does not implement StreamingJournal", limit, j.inner)
	}

	iter, err := streaming.ReadStreamFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, err
	}

	return &transformingIterator{inner: iter, sourceT: j.sourceT}, nil
}

// Close closes the inner journal when it implements io.Closer.
func (j *decoratedJournal) Close() error {
	if c, ok := j.inner.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

// transformingIterator adapts an EventIterator so that a batch SourceTransform
// is applied without materializing the whole journal. It pulls
// journalTransformChunkSize events at a time, transforms the chunk, and yields
// the transformed events one by one.
type transformingIterator struct {
	inner     EventIterator
	sourceT   SourceTransform
	pending   []Event
	exhausted bool
}

var _ EventIterator = (*transformingIterator)(nil)

// Next returns the next transformed event, or io.EOF when exhausted.
func (it *transformingIterator) Next() (Event, error) {
	for len(it.pending) == 0 {
		if it.exhausted {
			return nil, io.EOF
		}

		chunk := make([]Event, 0, journalTransformChunkSize)
		for len(chunk) < journalTransformChunkSize {
			evt, err := it.inner.Next()
			if errors.Is(err, io.EOF) {
				it.exhausted = true

				break
			}
			if err != nil {
				return nil, err
			}

			chunk = append(chunk, evt)
		}

		if len(chunk) == 0 {
			continue
		}

		transformed, err := it.sourceT(chunk)
		if err != nil {
			return nil, err
		}

		it.pending = transformed
	}

	evt := it.pending[0]
	it.pending = it.pending[1:]

	return evt, nil
}

// Close releases the transformed buffer and closes the inner iterator.
func (it *transformingIterator) Close() error {
	it.pending = nil
	it.exhausted = true

	return it.inner.Close()
}
