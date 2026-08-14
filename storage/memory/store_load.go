package memory

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Load returns all events for a stream.
// Returns ErrStreamNotFound if no events exist for the stream.
func (s *MemoryStore) Load(
	_ context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, "load")
	if err != nil {
		return nil, err
	}

	return events, nil
}

// LoadFromVersion returns events starting from the given version (exclusive). Returns a defensive copy.
// Returns ErrStreamNotFound if no events exist for the stream.
func (s *MemoryStore) LoadFromVersion(
	_ context.Context,
	ref id.StreamRef,
	version event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(ref, "load from version",
		func(evts []event.Event) []event.Event {
			return event.SliceFromVersion(evts, version)
		})
}

// LoadToVersion returns events up to and including maxVersion. Returns a defensive copy.
// Returns ErrStreamNotFound if no events exist for the stream.
func (s *MemoryStore) LoadToVersion(
	_ context.Context,
	ref id.StreamRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadFiltered(ref, "load to version",
		func(evts []event.Event) []event.Event {
			return event.SliceToVersion(evts, maxVersion)
		})
}

// LoadToTimestamp returns events where OccurredAt <= maxTime. Returns a defensive copy.
// Returns ErrStreamNotFound if no events exist for the stream.
func (s *MemoryStore) LoadToTimestamp(
	_ context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadFiltered(ref, "load to timestamp",
		func(evts []event.Event) []event.Event {
			return event.FilterByTimestamp(evts, maxTime)
		})
}

// LoadBackwards returns events in reverse version order (newest first).
// Returns ErrStreamNotFound if no events exist for the stream.
func (s *MemoryStore) LoadBackwards(
	_ context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, "load backwards")
	if err != nil {
		return nil, err
	}

	reversed := slices.Clone(events)
	slices.Reverse(reversed)

	return reversed, nil
}

// loadFiltered loads a stream's events and applies a filter function.
func (s *MemoryStore) loadFiltered(
	ref id.StreamRef,
	op string,
	filter func([]event.Event) []event.Event,
) ([]event.Event, error) {
	events, err := s.getEvents(ref, op)
	if err != nil {
		return nil, err
	}

	return filter(events), nil
}

func (s *MemoryStore) getEvents(
	ref id.StreamRef,
	op string,
) ([]event.Event, error) {
	return WithReadLock(s.LogStore,
		"memory.load_failed",
		fmt.Sprintf("memory store %s failed", op),
		func() ([]event.Event, error) {
			return s.LoadStreamLocked(op, ref.StreamKey(), nil)
		},
	)
}

func (s *MemoryStore) ReadAll(_ context.Context) ([]event.Event, error) {
	return WithReadLock(s.LogStore,
		"memory.read_all_failed",
		"memory store read all",
		func() ([]event.Event, error) {
			return s.ReadAllLocked(), nil
		},
	)
}

// ReadFrom retrieves events ordered by insertion order, starting after the given event ID.
// A missing start position replays from the beginning — safe for idempotent
// projections. Implements event.SeekableJournal for efficient projection catch-up.
func (s *MemoryStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	return WithReadLock(s.LogStore, "memory.read_from_failed",
		fmt.Sprintf("memory store read from (limit=%d) failed", limit),
		func() ([]event.Event, error) {
			return s.ReadFromLocked(afterEventID, limit, true), nil
		})
}
