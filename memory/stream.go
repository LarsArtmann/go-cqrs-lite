package memory

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// LoadStream is the streaming equivalent of Load.
// Since MemoryStore is in-memory, this delegates to Load and wraps the slice
// in a SliceIterator. This exists for interface conformance so consumers can
// type-assert to event.StreamingSource uniformly across store implementations.
func (s *MemoryStore) LoadStream(
	_ context.Context,
	ref event.AggregateRef,
) (event.EventIterator, error) {
	events, err := s.Load(context.Background(), ref)
	if err != nil {
		return nil, err
	}

	return event.NewSliceIterator(events), nil
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (s *MemoryStore) LoadStreamFromVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) (event.EventIterator, error) {
	events, err := s.LoadFromVersion(context.Background(), ref, version)
	if err != nil {
		return nil, err
	}

	return event.NewSliceIterator(events), nil
}

// ReadStream is the streaming equivalent of ReadAll.
func (s *MemoryStore) ReadStream(_ context.Context) (event.EventIterator, error) {
	events, err := s.ReadAll(context.Background())
	if err != nil {
		return nil, err
	}

	return event.NewSliceIterator(events), nil
}

// ReadStreamFrom is the streaming equivalent of ReadFrom.
func (s *MemoryStore) ReadStreamFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	events, err := s.ReadFrom(context.Background(), afterEventID, limit)
	if err != nil {
		return nil, err
	}

	return event.NewSliceIterator(events), nil
}

var (
	_ event.StreamingSource  = (*MemoryStore)(nil)
	_ event.StreamingJournal = (*MemoryStore)(nil)
)
