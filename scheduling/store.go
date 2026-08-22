package scheduling

import (
	"context"
	"sync"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TimerMarker is a phantom type for branding [TimerID].
type TimerMarker struct{}

// TimerID uniquely identifies a scheduled timer.
//
// It is string-backed (the id.StreamID pattern), NOT ULID-backed: timer IDs
// are semantic idempotency keys ("cancel-order-01J...", "delay-test") that
// callers choose for stable re-scheduling and cancellation. Forcing ULIDs
// would break every idempotent scheduling flow.
//
// Construct from a semantic name with [ParseTimerID].
type TimerID = cbid.ID[TimerMarker, string]

// ParseTimerID converts a semantic timer name into a [TimerID]. Accepts any
// non-empty string. Returns [ErrEmptyTimerID] for empty input.
func ParseTimerID(s string) (TimerID, error) {
	if s == "" {
		return TimerID{}, ErrEmptyTimerID
	}

	return cbid.NewID[TimerMarker](s), nil
}

// MustParseTimerID is [ParseTimerID] for compile-time-known timer names. It
// panics on empty input — a programming error, not a runtime condition.
func MustParseTimerID(s string) TimerID {
	timerID, err := ParseTimerID(s)
	if err != nil {
		panic(err)
	}

	return timerID
}

// Timer represents a scheduled command to fire at a future time.
//
// The type parameter P is the payload type delivered to the dispatch callback —
// typically a concrete command type (e.g. Timer[CancelOrderCmd]). Using a
// generic instead of `any` gives callers compile-time payload safety and clean
// JSON round-tripping, without forcing every consumer onto the same envelope.
type Timer[P any] struct {
	// ID is a unique identifier for this timer. If a timer with the same ID
	// already exists, Schedule is a no-op (idempotent scheduling).
	ID TimerID `json:"id"`

	// FireAt is when the timer should trigger. Must be in the future.
	FireAt time.Time `json:"fireAt"`

	// Payload is delivered to the dispatch callback when the timer fires.
	Payload P `json:"payload"`

	// Actor attributes the timer's eventual dispatch to an initiator.
	// Set it when the actor who scheduled the timer should remain the actor
	// of the command the timer later dispatches — the audit-trail answer to
	// "who caused this timeout to fire?".
	//
	// The zero value means unspecified: the dispatch callback decides
	// attribution (typically id.NewSystemActor("scheduler")). The JSON wire
	// form is the self-describing "kind:raw" string ("user:01JXYZ...",
	// "system:scheduler") — omitted when zero — exactly the shape the field
	// carried before it became typed.
	Actor id.ActorID `json:"actor,omitzero"`
}

// TimerStore persists scheduled timers across restarts.
type TimerStore[P any] interface {
	// Schedule records a timer. If a timer with the same ID already exists
	// and has not fired yet, it is a no-op (idempotent).
	Schedule(ctx context.Context, t Timer[P]) error

	// Due returns timers whose FireAt is at or before the given time,
	// ordered by FireAt ascending.
	Due(ctx context.Context, now time.Time) ([]Timer[P], error)

	// MarkFired removes a timer after it has been dispatched.
	MarkFired(ctx context.Context, id TimerID) error

	// Cancel removes a timer before it fires (e.g., order paid → cancel timeout).
	Cancel(ctx context.Context, id TimerID) error
}

// DispatchFunc is called when a timer fires. It receives the timer's payload.
// If it returns an error, the timer is re-scheduled for retry (up to MaxRetries).
type DispatchFunc[P any] func(ctx context.Context, t Timer[P]) error

// MemoryTimerStore is an in-memory TimerStore for development and testing.
type MemoryTimerStore[P any] struct {
	mu     sync.Mutex
	timers map[TimerID]Timer[P]
}

// NewMemoryTimerStore creates an in-memory timer store.
func NewMemoryTimerStore[P any]() *MemoryTimerStore[P] {
	return &MemoryTimerStore[P]{timers: make(map[TimerID]Timer[P])} //nolint:exhaustruct
}

func (s *MemoryTimerStore[P]) Schedule(_ context.Context, t Timer[P]) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.timers[t.ID]; exists {
		return nil
	}

	s.timers[t.ID] = t

	return nil
}

func (s *MemoryTimerStore[P]) Due(_ context.Context, now time.Time) ([]Timer[P], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []Timer[P]

	for _, t := range s.timers {
		if !t.FireAt.After(now) {
			due = append(due, t)
		}
	}

	return due, nil
}

func (s *MemoryTimerStore[P]) MarkFired(_ context.Context, id TimerID) error {
	return s.deleteTimer(id)
}

func (s *MemoryTimerStore[P]) Cancel(_ context.Context, id TimerID) error {
	return s.deleteTimer(id)
}

func (s *MemoryTimerStore[P]) deleteTimer(id TimerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.timers, id)

	return nil
}
