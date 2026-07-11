package projectionhost

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// WorkerStatus represents the lifecycle state of a projection worker.
type WorkerStatus string

const (
	// WorkerIdle means the worker is registered but Start has not been called.
	WorkerIdle WorkerStatus = "idle"
	// WorkerRunning means the worker is actively processing events.
	WorkerRunning WorkerStatus = "running"
	// WorkerLive means the worker caught up on replay and is processing live events.
	WorkerLive WorkerStatus = "live"
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
	Name       string        `json:"name"`
	Status     WorkerStatus  `json:"status"`
	Checkpoint string        `json:"checkpoint"`
	Processed  int64         `json:"processed"`
	Errors     int64         `json:"errors"`
	Restarts   int           `json:"restarts"`
	Lag        time.Duration `json:"lag"`
	LastError  string        `json:"lastError,omitempty"`
}

// DeadLetterEntry captures a poison message that exceeded the retry threshold.
// Event holds the original event so it can be replayed once the underlying
// handler bug is fixed; the string fields mirror the common diagnostics and
// survive JSON serialization even when Event is nil.
//
// This is the PROJECTION-side dead-letter: an event that poisoned a projection
// handler, captured for later replay. For the DISPATCH-side dead-letter
// (command/event/query that exhausted retries in the middleware pipeline),
// see middleware.DeadLetterEntry. The two types are intentionally separate —
// see ADR-0043 for the rationale.
type DeadLetterEntry struct {
	ProjectionName string
	EventID        string
	EventType      string
	AggregateID    string
	Event          event.Event // original poison event; needed for Replay
	Error          string
	ErrorCode      string // machine-readable code (e.g. "graph.sink.unknown_node_label"); empty if unclassified
	ErrorFamily    string // taxonomy family: "rejection", "conflict", etc.; empty if unclassified
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

// DeadLetterStoreAdmin extends DeadLetterStore with production management
// capabilities: counting, pagination, and time-bounded purge.
// SQLiteDeadLetterStore implements this interface; consumers can type-assert
// to access these features without requiring them from every implementation.
//
//	if admin, ok := store.(projectionhost.DeadLetterStoreAdmin); ok {
//	    count, _ := admin.Count(ctx)
//	}
type DeadLetterStoreAdmin interface {
	DeadLetterStore
	// Count returns the total number of dead-letter entries across all projections.
	Count(ctx context.Context) (int64, error)
	// ListPaged returns dead-letter entries with pagination.
	// An empty projectionName returns entries across all projections.
	// offset is zero-based; limit is the maximum entries to return (0 defaults to 50).
	ListPaged(
		ctx context.Context,
		projectionName string,
		offset, limit int,
	) ([]DeadLetterEntry, error)
	// PurgeBefore removes all entries that failed before the given timestamp.
	// Returns the number of entries removed.
	PurgeBefore(ctx context.Context, before time.Time) (int64, error)
}

var (
	_ DeadLetterStore      = (*MemoryDeadLetterStore)(nil)
	_ DeadLetterStoreAdmin = (*MemoryDeadLetterStore)(nil)
)

// MemoryDeadLetterStore is an in-memory DeadLetterStore for development and testing.
// Entries are lost on restart. For production use, prefer SQLiteDeadLetterStore,
// which persists entries across restarts.
type MemoryDeadLetterStore struct {
	mu      sync.Mutex
	entries []DeadLetterEntry
}

// NewMemoryDeadLetterStore creates an in-memory dead-letter store.
func NewMemoryDeadLetterStore() *MemoryDeadLetterStore {
	return &MemoryDeadLetterStore{} //nolint:exhaustruct // intentional zero-value init
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

func (s *MemoryDeadLetterStore) Count(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return int64(len(s.entries)), nil
}

func (s *MemoryDeadLetterStore) ListPaged(
	_ context.Context,
	projectionName string,
	offset, limit int,
) ([]DeadLetterEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []DeadLetterEntry

	for _, e := range s.entries {
		if projectionName == "" || e.ProjectionName == projectionName {
			filtered = append(filtered, e)
		}
	}

	if offset >= len(filtered) {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	end := min(offset+limit, len(filtered))

	result := make([]DeadLetterEntry, end-offset)
	copy(result, filtered[offset:end])

	return result, nil
}

func (s *MemoryDeadLetterStore) PurgeBefore(_ context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64

	filtered := s.entries[:0]
	for _, e := range s.entries {
		if e.FailedAt.Before(before) {
			count++

			continue
		}

		filtered = append(filtered, e)
	}

	s.entries = filtered

	return count, nil
}
