package projectionhost

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// WorkerStatus represents the lifecycle state of a projection worker.
type WorkerStatus string

const (
	// WorkerIdle means the worker is registered but Start has not been called.
	WorkerIdle WorkerStatus = "idle"
	// WorkerRunning means the worker is actively processing events.
	WorkerRunning WorkerStatus = "running"
	// WorkerBackoff means the worker is waiting before a restart.
	WorkerBackoff WorkerStatus = "backoff"
	// WorkerDraining means the worker is shutting down, waiting for in-flight events.
	WorkerDraining WorkerStatus = "draining"
	// WorkerStopped means the worker was gracefully stopped.
	WorkerStopped WorkerStatus = "stopped"
	// WorkerFailed means the worker exhausted its restart budget and gave up.
	WorkerFailed WorkerStatus = "failed"
)

// WorkerState is a point-in-time snapshot of a single worker's state.
type WorkerState struct {
	Name       string       `json:"name"`
	Status     WorkerStatus `json:"status"`
	Checkpoint string       `json:"checkpoint"`
	Processed  int64        `json:"processed"`
	Errors     int64        `json:"errors"`
	Restarts   int          `json:"restarts"`
	LastError  string       `json:"last_error,omitempty"`
}

// DeadLetterEntry captures a poison message that exceeded the retry threshold.
// Event holds the original event so it can be replayed once the underlying
// handler bug is fixed; the string fields mirror the common diagnostics and
// survive JSON serialization even when Event is nil.
type DeadLetterEntry struct {
	ProjectionName string
	EventID        string
	EventType      string
	AggregateID    string
	Event          event.Event // original poison event; needed for Replay
	Error          string
	FailedAt       time.Time
}

// DeadLetterStore stores poison messages for later inspection or replay.
type DeadLetterStore interface {
	// Store records a dead-letter entry.
	Store(ctx context.Context, entry DeadLetterEntry) error
	// List returns dead-letter entries for the given projection name.
	// An empty projectionName returns entries across all projections.
	List(ctx context.Context, projectionName string) ([]DeadLetterEntry, error)
	// Delete removes a single dead-letter entry by projection name and event ID.
	// Use it after a pure ReplayDeadLetters run to clear only the entries that
	// replayed successfully, leaving still-failing entries in place. Prefer
	// Delete over Purge when you want surgical cleanup.
	Delete(ctx context.Context, projectionName, eventID string) error
	// Purge removes ALL dead-letter entries for the given projection name.
	// An empty projectionName purges every entry across all projections.
	// Use Purge only when every entry for the projection has been resolved;
	// for partial cleanup after replay, use Delete per successful entry.
	Purge(ctx context.Context, projectionName string) error
}

// MemoryDeadLetterStore is an in-memory DeadLetterStore for development and testing.
type MemoryDeadLetterStore struct {
	mu      sync.Mutex
	entries []DeadLetterEntry
}

// NewMemoryDeadLetterStore creates an in-memory dead-letter store.
func NewMemoryDeadLetterStore() *MemoryDeadLetterStore {
	return &MemoryDeadLetterStore{}
}

func (s *MemoryDeadLetterStore) Store(_ context.Context, entry DeadLetterEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)

	return nil
}

func (s *MemoryDeadLetterStore) List(
	_ context.Context,
	projectionName string,
) ([]DeadLetterEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if projectionName == "" {
		result := make([]DeadLetterEntry, len(s.entries))
		copy(result, s.entries)

		return result, nil
	}

	var result []DeadLetterEntry

	for _, e := range s.entries {
		if e.ProjectionName == projectionName {
			result = append(result, e)
		}
	}

	return result, nil
}

func (s *MemoryDeadLetterStore) Delete(_ context.Context, projectionName, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.entries[:0]
	for _, e := range s.entries {
		// Keep entries that do not match BOTH the projection and the event ID.
		// This makes Delete safe to call with a non-empty projectionName to
		// scope the delete, or with an empty one to delete by eventID globally.
		if e.ProjectionName == projectionName && e.EventID == eventID {
			continue
		}

		filtered = append(filtered, e)
	}

	s.entries = filtered

	return nil
}

func (s *MemoryDeadLetterStore) Purge(_ context.Context, projectionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if projectionName == "" {
		s.entries = nil

		return nil
	}

	filtered := s.entries[:0]
	for _, e := range s.entries {
		if e.ProjectionName != projectionName {
			filtered = append(filtered, e)
		}
	}

	s.entries = filtered

	return nil
}
