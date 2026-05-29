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
	events, err := s.getEvents(aggregateType, aggregateID, "load")
	if err != nil {
		return nil, err
	}

	return copyEvents(events), nil
}

// loadFiltered is a shared helper that loads events for an aggregate and applies a filter function.
func (s *MemoryStore) loadFiltered(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	op string,
	filter func([]event.Event) []event.Event,
) ([]event.Event, error) {
	events, err := s.getEvents(aggregateType, aggregateID, op)
	if err != nil {
		return nil, err
	}

	return copyEvents(filter(events)), nil
}

// LoadFromVersion returns events starting from the given version (exclusive). Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(aggregateType, aggregateID, "load from version", func(evts []event.Event) []event.Event {
		return event.SliceFromVersion(evts, version)
	})
}

// LoadToVersion returns events up to and including maxVersion. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(aggregateType, aggregateID, "load to version", func(evts []event.Event) []event.Event {
		return event.SliceToVersion(evts, maxVersion)
	})
}

// LoadToTimestamp returns events where OccurredAt <= maxTime. Returns a defensive copy.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *MemoryStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadFiltered(aggregateType, aggregateID, "load to timestamp", func(evts []event.Event) []event.Event {
		return event.FilterByTimestamp(evts, maxTime)
	})
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
		return nil, fmt.Errorf("aggregate %s/%s: %w", aggregateType, aggregateID, event.ErrAggregateNotFound)
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

	reversed := slices.Clone(events)
	slices.Reverse(reversed)

	return reversed, nil
}

func copyEvents(events []event.Event) []event.Event {
	return slices.Clone(events)
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

	all := s.collectAllSorted()

	startIdx := 0

	if !afterEventID.IsZero() {
		if idx := slices.IndexFunc(all, func(e event.Event) bool { return e.ID() == afterEventID }); idx >= 0 {
			startIdx = idx + 1
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
