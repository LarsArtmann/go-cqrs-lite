package idempotency

import (
	"time"

	goidempotency "github.com/larsartmann/go-idempotency"
)

// Store is the idempotency key store interface.
type Store = goidempotency.Store

// MemoryStore is an in-memory implementation of Store.
type MemoryStore = goidempotency.MemoryStore

// ErrDuplicate is returned when a duplicate idempotency key is recorded.
var ErrDuplicate = goidempotency.ErrDuplicate

// NewMemoryStore creates a new in-memory idempotency store with a background
// sweep goroutine.
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore {
	return goidempotency.NewMemoryStore(sweepInterval)
}
