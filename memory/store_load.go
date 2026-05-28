package memory

import (
	"context"
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
	events, err := s.getEvents(aggregateType, aggregateID, "load")
	if err != nil {
		return nil, err
	}

	return copyEvents(events), nil
}

// LoadFromVersion returns events starting from the given version (exclusive). Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	events, err := s.getEvents(aggregateType, aggregateID, "load from version")
	if err != nil {
		return nil, err
	}

	if version.Int() >= len(events) {
		return []event.Event{}, nil
	}

	return copyEvents(events[version.Int():]), nil
}

// LoadToVersion returns events up to and including maxVersion. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	events, err := s.getEvents(aggregateType, aggregateID, "load to version")
	if err != nil {
		return nil, err
	}

	end := min(maxVersion.Int(), len(events))

	return copyEvents(events[:end]), nil
}

// LoadToTimestamp returns events where OccurredAt <= maxTime. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	events, err := s.getEvents(aggregateType, aggregateID, "load to timestamp")
	if err != nil {
		return nil, err
	}

	var filtered []event.Event

	for _, e := range events {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	return copyEvents(filtered), nil
}

func (s *MemoryStore) getEvents(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	op string,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.Wrapf(err, event.Infrastructure, "memory.load_failed", "memory store %s failed", op)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)

	events, exists := s.events[key]
	if !exists {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadBackwards returns events in reverse version order (newest first).
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadBackwards(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	events, err := s.getEvents(aggregateType, aggregateID, "load backwards")
	if err != nil {
		return nil, err
	}

	reversed := make([]event.Event, len(events))
	for i, e := range events {
		reversed[len(events)-1-i] = e
	}

	return reversed, nil
}

func copyEvents(events []event.Event) []event.Event {
	result := make([]event.Event, len(events))
	copy(result, events)

	return result
}

func (s *MemoryStore) collectAllSorted() []event.Event {
	var all []event.Event

	for _, events := range s.events {
		all = append(all, events...)
	}

	slices.SortFunc(all, func(a, b event.Event) int {
		return a.OccurredAt().Compare(b.OccurredAt())
	})

	return all
}

// ReadAll returns all events across all aggregates, sorted by OccurredAt.
// Implements event.Journal for projection replay.
func (s *MemoryStore) ReadAll(_ context.Context) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "memory.read_all_failed", "memory store read all")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.collectAllSorted()

	return copyEvents(all), nil
}

// LoadAll returns all events across all aggregates, sorted by OccurredAt.
//
// Deprecated: use ReadAll instead.
func (s *MemoryStore) LoadAll(ctx context.Context) ([]event.Event, error) {
	return s.ReadAll(ctx)
}

// ReadFrom retrieves events ordered by OccurredAt, starting after the given event ID.
// Implements event.SeekableJournal for efficient projection catch-up.
func (s *MemoryStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	err := s.CheckClosed(event.ErrStoreClosed)
	if err != nil {
		return nil, event.Wrapf(err, event.Infrastructure, "memory.read_from_failed", "memory store read from (limit=%d) failed", limit)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.collectAllSorted()

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

	return copyEvents(filtered), nil
}

// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after the given event ID.
//
// Deprecated: use ReadFrom instead.
func (s *MemoryStore) LoadAllFromPosition(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	return s.ReadFrom(ctx, afterEventID, limit)
}
