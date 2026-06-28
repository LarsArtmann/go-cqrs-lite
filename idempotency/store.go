package idempotency

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrDuplicate is returned by [Store.CheckAndRecord] when the key was already
// recorded and has not expired. It is classified as a Conflict: a retried
// command with the same idempotency key conflicts with a prior, still-valid
// recording.
var ErrDuplicate = event.NewConflict(
	"idempotency.duplicate",
	"key has already been recorded",
)

// Store tracks opaque keys (typically command idempotency keys) to prevent
// duplicate processing. When a key is seen for the first time, the store
// records it with a TTL. Subsequent lookups for the same key report it as seen
// until the TTL expires.
//
// This is essential for at-least-once delivery: if a client submits a command,
// loses the acknowledgement, and retries, the store prevents the command from
// executing twice.
//
// Implementations must be safe for concurrent use.
type Store interface {
	// Seen reports whether the key is currently recorded and not expired.
	Seen(ctx context.Context, key string) (bool, error)

	// Record marks the key as seen with the given TTL. If the key is already
	// recorded, it is a no-op (the TTL is not extended).
	Record(ctx context.Context, key string, ttl time.Duration) error

	// CheckAndRecord atomically reports whether the key was already seen and,
	// if not, records it. Returns [ErrDuplicate] if the key was already
	// recorded and not expired.
	//
	// Implementations MUST make this atomic (single lock or single round-trip)
	// to prevent the TOCTOU race that a separate Seen + Record pair would
	// create. For [MemoryStore] this is a single mutex; for a future Redis
	// store it would be a SET NX command; for SQL an INSERT ... ON CONFLICT
	// DO NOTHING.
	CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}

// MemoryStore is an in-memory [Store] with TTL-based expiration. It runs an
// optional background goroutine that sweeps expired entries. Safe for
// concurrent use.
//
// Expired entries are also deleted lazily on read, so the map cannot grow
// unboundedly even when the sweep goroutine is disabled (sweepInterval == 0).
type MemoryStore struct {
	mu       sync.RWMutex
	entries  map[string]time.Time // key → expiresAt
	stop     chan struct{}
	stopOnce sync.Once
}

// NewMemoryStore creates an in-memory idempotency store and, when sweepInterval
// is positive, starts a background goroutine that sweeps expired entries every
// sweepInterval. Call [MemoryStore.Close] to stop the sweeper.
//
// Pass sweepInterval == 0 to disable the background sweep; lazy deletion on
// read still bounds growth.
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore {
	s := &MemoryStore{
		entries: make(map[string]time.Time),
		stop:    make(chan struct{}),
	}
	if sweepInterval > 0 {
		go s.sweep(sweepInterval)
	}

	return s
}

// Seen reports whether the key is currently recorded and not expired.
// Expired entries are lazily deleted on read.
func (s *MemoryStore) Seen(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.entries[key]
	if !ok {
		return false, nil
	}

	if time.Now().Before(exp) {
		return true, nil
	}

	delete(s.entries, key)

	return false, nil
}

// Record marks the key as seen with the given TTL. If the key is already
// recorded, it is a no-op (the existing expiry is not extended).
func (s *MemoryStore) Record(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[key]; !ok {
		s.entries[key] = time.Now().Add(ttl)
	}

	return nil
}

// CheckAndRecord atomically claims a key. Returns [ErrDuplicate] if the key was
// already recorded and not expired. The check and the record happen under a
// single write lock, so concurrent callers with the same key are serialized:
// exactly one wins.
func (s *MemoryStore) CheckAndRecord(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exp, ok := s.entries[key]; ok && time.Now().Before(exp) {
		return ErrDuplicate
	}

	s.entries[key] = time.Now().Add(ttl)

	return nil
}

// Close stops the background sweep goroutine. Safe to call multiple times.
func (s *MemoryStore) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *MemoryStore) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now := time.No
			w()

			s.mu.Lock()
			for key, exp := range s.entries {
				if now.After(exp) {
					delete(s.entries, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
