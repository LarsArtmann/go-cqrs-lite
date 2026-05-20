package memory

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Load returns all events for an aggregate. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	result := make([]event.Event, len(events))
	copy(result, events)

	return result, nil
}

// LoadFromVersion returns events starting from the given version (exclusive). Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load from version: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	if version.Int() >= len(events) {
		return []event.Event{}, nil
	}

	sub := events[version.Int():]
	result := make([]event.Event, len(sub))
	copy(result, sub)

	return result, nil
}

// LoadToVersion returns events up to and including maxVersion. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load to version: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	end := min(maxVersion.Int(), len(events))

	result := make([]event.Event, end)
	copy(result, events[:end])

	return result, nil
}

// LoadToTimestamp returns events where OccurredAt <= maxTime. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load to timestamp: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := streamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	var filtered []event.Event

	for _, e := range events {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	result := make([]event.Event, len(filtered))
	copy(result, filtered)

	return result, nil
}

// LoadAll returns all events across all aggregates, sorted by OccurredAt.
// Implements event.GlobalLoader for projection replay.
func (s *MemoryStore) LoadAll(_ context.Context) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load all: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []event.Event

	for _, events := range s.events {
		all = append(all, events...)
	}

	slices.SortFunc(all, func(a, b event.Event) int {
		return a.OccurredAt().Compare(b.OccurredAt())
	})

	result := make([]event.Event, len(all))
	copy(result, all)

	return result, nil
}

// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after the given event ID.
// Implements event.PositionalLoader for efficient projection catch-up.
func (s *MemoryStore) LoadAllFromPosition(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, fmt.Errorf("memory store load all from position: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []event.Event

	for _, events := range s.events {
		all = append(all, events...)
	}

	slices.SortFunc(all, func(a, b event.Event) int {
		return a.OccurredAt().Compare(b.OccurredAt())
	})

	startIdx := 0

	if !afterEventID.IsZero() {
		for i, e := range all {
			if e.ID() == afterEventID {
				startIdx = i + 1

				break
			}
		}
	}

	filtered := all[startIdx:]
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	result := make([]event.Event, len(filtered))
	copy(result, filtered)

	return result, nil
}
