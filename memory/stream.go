package memory

import (
	"context"
	
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

var _ event.StreamLoader = (*MemoryStore)(nil)

// LoadStream returns a stream of events for a single aggregate, ordered by version.
// Implements event.StreamLoader for memory-efficient iteration over large aggregates.
func (s *MemoryStore) LoadStream(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (event.EventStream, error) {
	events, err := s.getEvents(aggregateType, aggregateID, "load stream")
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.load_stream_failed", "memory store load stream")
	}

	return event.NewSliceStream(copyEvents(events)), nil
}
