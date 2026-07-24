package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DeadLetterEntry captures a message that exhausted all retry attempts.
// It contains enough context to inspect, replay, or discard the failed
// message without access to the original handler.
//
// This is the DISPATCH-side dead-letter: a command/event/query that failed
// in the middleware retry pipeline. For the PROJECTION-side dead-letter
// (poison events captured during projection processing, with replay support),
// see projectionhost.DeadLetterEntry. The two types are intentionally
// separate — see ADR-0043 for the rationale.
type DeadLetterEntry struct {
	// Kind is the message category: "command", "event", or "query".
	Kind string

	// Type is the message type name (e.g., "user.created", "todo.create").
	Type string

	// StreamID is the stream the message targets. Zero for queries.
	StreamID id.StreamID

	// Error is the last error that caused retry exhaustion.
	Error error

	// ErrorCode is the machine-readable error code extracted from Error
	// (e.g. "event.version_conflict"). Empty if Error was unclassified.
	ErrorCode string

	// ErrorFamily is the taxonomy family: "rejection", "conflict", etc.
	// Empty if Error was unclassified.
	ErrorFamily string

	// Attempts is the total number of attempts made (including the first).
	Attempts int

	// FailedAt is when the message was moved to the dead-letter store.
	FailedAt time.Time
}

// DeadLetterHandler is called when a message exhausts all retry attempts.
// Use it to quarantine poison messages for inspection, logging, or replay.
//
// The handler must not panic — if it fails, the message is already lost
// (retries exhausted). Keep the handler fast and side-effect-minimal.
type DeadLetterHandler func(ctx context.Context, entry DeadLetterEntry)

// MemoryDeadLetterStore is an in-memory dead-letter handler for testing
// and single-process development. It is safe for concurrent use.
//
// Usage:
//
//	store := middleware.NewMemoryDeadLetterStore()
//	config := middleware.RetryConfig{
//	    MaxAttempts:  3,
//	    OnDeadLetter: store.Handle,
//	}
//	// ... run commands/events through the retry middleware ...
//	entries := store.Entries() // inspect dead-lettered messages
type MemoryDeadLetterStore struct {
	mu      sync.Mutex
	entries []DeadLetterEntry
}

// NewMemoryDeadLetterStore creates an empty in-memory dead-letter store.
func NewMemoryDeadLetterStore() *MemoryDeadLetterStore {
	return &MemoryDeadLetterStore{ //nolint:exhaustruct // mu is zero-value (unlocked)
		entries: make([]DeadLetterEntry, 0),
	}
}

// Handle stores a dead-letter entry. Implements DeadLetterHandler.
func (s *MemoryDeadLetterStore) Handle(_ context.Context, entry DeadLetterEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)
}

// Entries returns a copy of all dead-lettered messages.
func (s *MemoryDeadLetterStore) Entries() []DeadLetterEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]DeadLetterEntry(nil), s.entries...)
}

// Count returns the number of dead-lettered messages.
func (s *MemoryDeadLetterStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.entries)
}

// Clear removes all dead-lettered messages.
func (s *MemoryDeadLetterStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = s.entries[:0]
}
