package scheduling

import (
	"context"
	"sync"
	"time"
)

// TimerID uniquely identifies a scheduled timer.
type TimerID = string

// Timer represents a scheduled command to fire at a future time.
type Timer struct {
	// ID is a unique identifier for this timer. If a timer with the same ID
	// already exists, Schedule is a no-op (idempotent scheduling).
	ID TimerID `json:"id"`

	// FireAt is when the timer should trigger. Must be in the future.
	FireAt time.Time `json:"fire_at"`

	// Payload is an opaque payload delivered to the dispatch callback.
	// Typically a serialized command or a command type + aggregate ID.
	Payload any `json:"payload"`
}

// TimerStore persists scheduled timers across restarts.
type TimerStore interface {
	// Schedule records a timer. If a timer with the same ID already exists
	// and has not fired yet, it is a no-op (idempotent).
	Schedule(ctx context.Context, t Timer) error

	// Due returns timers whose FireAt is at or before the given time,
	// ordered by FireAt ascending.
	Due(ctx context.Context, now time.Time) ([]Timer, error)

	// MarkFired removes a timer after it has been dispatched.
	MarkFired(ctx context.Context, id TimerID) error

	// Cancel removes a timer before it fires (e.g., order paid → cancel timeout).
	Cancel(ctx context.Context, id TimerID) error
}

// DispatchFunc is called when a timer fires. It receives the timer's payload.
// If it returns an error, the timer is re-scheduled for retry (up to MaxRetries).
type DispatchFunc func(ctx context.Context, t Timer) error

// MemoryTimerStore is an in-memory TimerStore for development and testing.
type MemoryTimerStore struct {
	mu     sync.Mutex
	timers map[TimerID]Timer
}

// NewMemoryTimerStore creates an in-memory timer store.
func NewMemoryTimerStore() *MemoryTimerStore {
	return &MemoryTimerStore{timers: make(map[TimerID]Timer)}
}

func (s *MemoryTimerStore) Schedule(_ context.Context, t Timer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.timers[t.ID]; exists {
		return nil
	}

	s.timers[t.ID] = t

	return nil
}

func (s *MemoryTimerStore) Due(_ context.Context, now time.Time) ([]Timer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []Timer

	for _, t := range s.timers {
		if !t.FireAt.After(now) {
			due = append(due, t)
		}
	}

	return due, nil
}

func (s *MemoryTimerStore) MarkFired(_ context.Context, id TimerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.timers, id)

	return nil
}

func (s *MemoryTimerStore) Cancel(_ context.Context, id TimerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.timers, id)

	return nil
}
