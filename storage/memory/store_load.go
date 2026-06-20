package memory

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Load returns all events for an aggregate.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) Load(
	_ context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, "load")
	if err != nil {
		return nil, err
	}

	return events, nil
}

// loadFiltered is a shared helper that loads events for an aggregate and applies a filter function.
func (s *MemoryStore) loadFiltered(
	ref event.AggregateRef,
	op string,
	filter func([]event.Event) []event.Event,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, op)
	if err != nil {
		return nil, event.Wrapf(err, event.Classify(err), "memory.load_filtered", "op=%s", op)
	}

	return filter(events), nil
}

// LoadFromVersion returns events starting from the given version (exclusive). Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(
		ref,
		"load from version",
		func(evts []event.Event) []event.Event {
			return event.SliceFromVersion(evts, version)
		},
	)
}

// LoadToVersion returns events up to and including maxVersion. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToVersion(
	_ context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(
		ref,
		"load to version",
		func(evts []event.Event) []event.Event {
			return event.SliceToVersion(evts, maxVersion)
		},
	)
}

// LoadToTimestamp returns events where OccurredAt <= maxTime. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToTimestamp(
	_ context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadFiltered(
		ref,
		"load to timestamp",
		func(evts []event.Event) []event.Event {
			return event.FilterByTimestamp(evts, maxTime)
		},
	)
}

func (s *MemoryStore) getEvents(
	ref event.AggregateRef,
	op string,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.Wrapf(
			err,
			event.Infrastructure,
			"memory.load_failed",
			"memory store %s failed",
			op,
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	indices, exists := s.streamIndex[key]
	if !exists {
		return nil, event.WrapRejection(event.ErrAggregateNotFound,
			"memory.aggregate_not_found",
			fmt.Sprintf("memory %s aggregate %s not found", op, ref))
	}

	events := make([]event.Event, len(indices))
	for i, idx := range indices {
		events[i] = s.globalLog[idx]
	}

	return events, nil
}

// LoadBackwards returns events in reverse version order (newest first).
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadBackwards(
	_ context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, "load backwards")
	if err != nil {
		return nil, err
	}

	reversed := slices.Clone(events)
	slices.Reverse(reversed)

	return reversed, nil
}

func copyEvents(events []event.Event) []event.Event {
	return slices.Clone(events)
}

func (s *MemoryStore) ReadAll(_ context.Context) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.read_all_failed", "memory store read all")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return copyEvents(s.globalLog), nil
}

// ReadFrom retrieves events ordered by insertion order, starting after the given event ID.
// Implements event.SeekableJournal for efficient projection catch-up.
func (s *MemoryStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.Wrapf(
			err,
			event.Infrastructure,
			"memory.read_from_failed",
			"memory store read from (limit=%d) failed",
			limit,
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	startIdx := 0

	if !afterEventID.IsZero() {
		if idx, ok := s.eventIDIndex[afterEventID]; ok {
			startIdx = idx + 1
		}
	}

	filtered := s.globalLog[startIdx:]
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return copyEvents(filtered), nil
}
