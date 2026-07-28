package memory

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// sliceIteratorOrErr wraps the "load events, on error return wrapped err,
// else return SliceIterator" pattern shared by all streaming LoadX/ReadX
// methods on MemoryStore.
func sliceIteratorOrErr(events []event.Event, err error, code, msg string) (event.EventIterator, error) {
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, code, msg)
	}

	return event.NewSliceIterator(events), nil
}

// LoadStream is the streaming equivalent of Load.
// Since MemoryStore is in-memory, this delegates to Load and wraps the slice
// in a SliceIterator. This exists for interface conformance so consumers can
// type-assert to event.StreamingSource uniformly across store implementations.
func (s *MemoryStore) LoadStream(
	ctx context.Context,
	ref id.StreamRef,
) (event.EventIterator, error) {
	events, err := s.Load(ctx, ref)

	return sliceIteratorOrErr(events, err, "memory.load_stream", "load events for stream")
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (s *MemoryStore) LoadStreamFromVersion(
	ctx context.Context,
	ref id.StreamRef,
	version event.Version,
) (event.EventIterator, error) {
	events, err := s.LoadFromVersion(ctx, ref, version)

	return sliceIteratorOrErr(events, err, "memory.load_stream_from_version",
		"load events from version for stream")
}

// ReadStream is the streaming equivalent of ReadAll.
func (s *MemoryStore) ReadStream(ctx context.Context) (event.EventIterator, error) {
	events, err := s.ReadAll(ctx)

	return sliceIteratorOrErr(events, err, "memory.read_stream", "read all events for stream")
}

// ReadStreamFrom is the streaming equivalent of ReadFrom.
func (s *MemoryStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	events, err := s.ReadFrom(ctx, afterEventID, limit)

	return sliceIteratorOrErr(events, err, "memory.read_stream_from",
		"read events from position for stream")
}

var (
	_ event.StreamingSource  = (*MemoryStore)(nil)
	_ event.StreamingJournal = (*MemoryStore)(nil)
)
