package event

import (
	"context"
	"io"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// EventIterator yields events one at a time, avoiding the need to
// materialize the full result set into a slice. Iterators must be closed
// to release resources (file handles, DB cursors, network buffers).
//
// Usage:
//
//	iter, err := store.LoadStream(ctx, ref)
//	if err != nil { return err }
//	defer iter.Close()
//
//	for {
//	    evt, err := iter.Next()
//	    if err == io.EOF { break }
//	    if err != nil { return err }
//	    handle(evt)
//	}
//
// Implementations are NOT goroutine-safe — each iterator is single-threaded.
type EventIterator interface {
	// Next returns the next event, or io.EOF when no more events remain.
	// Any other error indicates a read failure; the iterator should be closed.
	Next() (Event, error)

	// Close releases resources held by the iterator. Safe to call multiple times.
	// After Close, Next returns io.EOF.
	Close() error
}

// StreamingSource extends EventSource with streaming reads that avoid
// materializing full slices. Useful for streams with large event histories.
//
// This is an opt-in interface — existing stores can implement it by wrapping
// their slice-returning methods with NewSliceIterator.
type StreamingSource interface {
	// LoadStream is the streaming equivalent of Load.
	LoadStream(ctx context.Context, ref id.StreamRef) (EventIterator, error)

	// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
	LoadStreamFromVersion(
		ctx context.Context,
		ref id.StreamRef,
		version Version,
	) (EventIterator, error)
}

// StreamingJournal extends Journal with streaming reads that avoid
// materializing the full journal into memory. Essential for large event stores
// where ReadAll would OOM.
//
// This is an opt-in interface.
type StreamingJournal interface {
	// ReadStream is the streaming equivalent of ReadAll.
	ReadStream(ctx context.Context) (EventIterator, error)

	// ReadStreamFrom is the streaming equivalent of ReadFrom.
	ReadStreamFrom(
		ctx context.Context,
		afterEventID id.EventID,
		limit int,
	) (EventIterator, error)
}

// SliceIterator adapts a pre-loaded []Event slice to the EventIterator
// interface. Useful for stores that load slices internally but want to
// satisfy the streaming interfaces without code duplication.
//
//	iter := event.NewSliceIterator(events)
//	defer iter.Close()
type SliceIterator struct {
	events []Event
	pos    int
	closed bool
}

// NewSliceIterator creates an iterator over a pre-loaded slice.
// The iterator does not copy the slice — callers must not modify it
// while the iterator is active.
func NewSliceIterator(events []Event) *SliceIterator {
	return &SliceIterator{events: events}
}

// Next returns the next event, or io.EOF when exhausted or closed.
func (s *SliceIterator) Next() (Event, error) {
	if s.closed || s.pos >= len(s.events) {
		return nil, io.EOF
	}

	evt := s.events[s.pos]
	s.pos++

	return evt, nil
}

// Close marks the iterator as exhausted. Subsequent Next calls return io.EOF.
func (s *SliceIterator) Close() error {
	s.closed = true
	s.events = nil
	return nil
}
