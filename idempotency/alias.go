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

// ErrInvalidTTL is returned by Record and CheckAndRecord when the TTL is not
// positive. A non-positive TTL records an expiry already in the past, so the
// key protects nothing.
var ErrInvalidTTL = goidempotency.ErrInvalidTTL

// NewMemoryStore creates a new in-memory idempotency store with a background
// sweep goroutine.
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore {
	return goidempotency.NewMemoryStore(sweepInterval)
}
