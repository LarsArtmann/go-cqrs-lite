package memory

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// LoadStream is the streaming equivalent of Load.
// Since MemoryStore is in-memory, this delegates to Load and wraps the slice
// in a SliceIterator. This exists for interface conformance so consumers can
// type-assert to event.StreamingSource uniformly across store implementations.
func (s *MemoryStore) LoadStream(
	ctx context.Context,
	ref event.AggregateRef,
) (event.EventIterator, error) {
	events, err := s.Load(ctx, ref)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.load_stream",
			"load events for stream")
	}

	return event.NewSliceIterator(events), nil
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (s *MemoryStore) LoadStreamFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) (event.EventIterator, error) {
	events, err := s.LoadFromVersion(ctx, ref, version)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.load_stream_from_version",
			"load events from version for stream")
	}

	return event.NewSliceIterator(events), nil
}

// ReadStream is the streaming equivalent of ReadAll.
func (s *MemoryStore) ReadStream(ctx context.Context) (event.EventIterator, error) {
	events, err := s.ReadAll(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.read_stream",
			"read all events for stream")
	}

	return event.NewSliceIterator(events), nil
}

// ReadStreamFrom is the streaming equivalent of ReadFrom.
func (s *MemoryStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	events, err := s.ReadFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.read_stream_from",
			"read events from position for stream")
	}

	return event.NewSliceIterator(events), nil
}

var (
	_ event.StreamingSource  = (*MemoryStore)(nil)
	_ event.StreamingJournal = (*MemoryStore)(nil)
)
