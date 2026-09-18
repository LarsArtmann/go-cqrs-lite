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

// AsOfSignal documents the query-input convention for temporal reads. The
// marker itself is retained for documentation and type-assertion purposes;
// the ACTIVE mechanism (ADR-0141 §4) is the input struct convention:
// declare an `AsOf time.Time` field on a point-lookup input —
//
//	type AccountBalance struct {
//	    AccountID string
//	    AsOf      time.Time  // non-zero → temporal read via VersionedStorage
//	}
//
// A non-zero value routes the read through the engine's VersionedStorage
// capability; the zero value means "latest" (the normal path). The explicit
// named form is [Store.ExecuteAsOf].
type AsOfSignal struct {
	Timestamp time.Time
}

// unsupportedEngineVersioned is the shared error for temporal reads landing
// on an engine without the VersionedStorage capability. Failing loud beats
// silently degrading to a latest-only read (ADR-0141 §5).
func unsupportedEngineVersioned(eng Engine) error {
	return fmt.Errorf("%w: engine %s does not support versioned reads",
		ErrUnsupportedADT, eng.Profile().Name)
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

	vs, ok := q.QueryEngine().(VersionedStorage)
	if !ok || !EngineVersionsCells(q.QueryEngine()) {
		return nil, unsupportedEngineVersioned(q.QueryEngine())
	}

	val, err := vs.MapGetAsOf(ctx, collection, key, t)
	if err != nil {
		return nil, fmt.Errorf("ExecuteAsOf: %w", err)
	}

	return val, nil
}
