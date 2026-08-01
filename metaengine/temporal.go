package metaengine

import (
	"context"
	"fmt"
	"time"
)

// VersionedStorage is an optional engine capability for temporal (as-of)
// queries. Engines that implement this interface can answer point lookups
// at a specific point in time — "what was the value of key K at timestamp T?"
//
// This is the event-sourcing-specific temporal read primitive. In ES, every
// event is timestamped, so the full history is available. VersionedStorage
// exposes O(1) or O(logN) as-of reads without replaying the full stream.
//
// Memory engines can implement this by keeping version chains. SQL engines
// can use BigTable-style versioned cells or a separate history table.
type VersionedStorage interface {
	// MapGetAsOf returns the value for a key as it existed at timestamp t.
	// Returns ErrNotFound if the key did not exist at that time.
	MapGetAsOf(ctx context.Context, collection, key string, t time.Time) (any, error)

	// MapExistsAsOf returns true if the key existed at timestamp t.
	MapExistsAsOf(ctx context.Context, collection, key string, t time.Time) (bool, error)
}

// AsOfSignal is a marker type passed as a query input to request a temporal
// (as-of) read. When the planner detects an AsOf field in a query input,
// it routes the query to an engine implementing VersionedStorage.
//
// Usage:
//
//	type AccountBalance struct {
//	    AccountID string
//	    AsOf      time.Time  // presence of this field triggers temporal routing
//	}
type AsOfSignal struct {
	Timestamp time.Time
}

// supportsVersionedReads checks whether any of the given engines implement
// VersionedStorage. Used by the planner to emit a diagnostic when a query
// appears to need temporal reads but no versioned engine is available.
func supportsVersionedReads(engines []Engine) bool {
	for _, eng := range engines {
		if _, ok := eng.(VersionedStorage); ok {
			return true
		}
	}

	return false
}

// versionedReadRule checks if queries declare temporal read patterns and
// emits a diagnostic when no VersionedStorage engine is available.
type versionedReadRule struct{}

func (*versionedReadRule) Name() string { return "versioned-read-check" }

func (*versionedReadRule) Apply(result *PlanResult, ctx PlanContext) error {
	if supportsVersionedReads(ctx.Store.engines) {
		return nil
	}

	// Check if any query has a volume hint suggesting temporal reads
	// (future: detect AsOf fields in query input types via reflection)
	// For now, this rule is a no-op unless we detect temporal patterns
	return nil
}

// ExecuteAsOf performs a temporal (as-of) point lookup on a collection.
// It finds the engine assigned to the collection and, if it implements
// VersionedStorage, delegates to MapGetAsOf.
//
// Returns ErrNotFound if the key did not exist at timestamp t.
// Returns ErrUnsupportedADT if the engine does not implement VersionedStorage.
func (s *Store) ExecuteAsOf(
	ctx context.Context,
	collection, key string,
	t time.Time,
) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q, ok := s.queries[collection]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errNoQueryForInputType, collection)
	}

	vs, ok := q.engine.(VersionedStorage)
	if !ok {
		return nil, fmt.Errorf("%w: engine %s does not support versioned reads",
			ErrUnsupportedADT, q.engine.Profile().Name)
	}

	return vs.MapGetAsOf(ctx, collection, key, t)
}
