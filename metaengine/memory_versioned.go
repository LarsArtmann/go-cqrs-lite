package metaengine

import (
	"context"
	"sort"
	"time"
)

// versionedEntry records a single value at a point in time.
type versionedEntry struct {
	ts    time.Time
	value any // nil means the key was deleted at this timestamp
}

// versionChain stores the append-only history of a single key. Entries are
// ordered by timestamp (ascending) because time.Now() is monotonic within
// a single process. Binary search finds the latest entry <= t.
type versionChain struct {
	entries []versionedEntry
}

// asOf returns the value and existence at timestamp t. The value is the
// latest entry with ts <= t. If that entry has value == nil, the key was
// deleted at that time and asOf returns (nil, false).
func (vc *versionChain) asOf(t time.Time) (any, bool) {
	idx := sort.Search(len(vc.entries), func(i int) bool {
		return vc.entries[i].ts.After(t)
	})

	if idx == 0 {
		return nil, false // no entries before t
	}

	entry := vc.entries[idx-1] // latest entry <= t

	if entry.value == nil {
		return nil, false // was deleted before t
	}

	return entry.value, true
}

// recordVersion appends a timestamped entry to the key's version chain.
// Caller MUST hold m.mu.Lock().
func (m *memoryEngine) recordVersion(col, key string, value any) {
	if m.versions[col] == nil {
		m.versions[col] = make(map[string]*versionChain)
	}

	chain, ok := m.versions[col][key]
	if !ok {
		chain = &versionChain{}
		m.versions[col][key] = chain
	}

	chain.entries = append(chain.entries, versionedEntry{
		ts:    time.Now(),
		value: value,
	})
}

// --- VersionedStorage implementation ---

// MapGetAsOf returns the value for a key as it existed at timestamp t.
// Returns ErrNotFound if the key did not exist at that time.
func (m *memoryEngine) MapGetAsOf(
	_ context.Context,
	col, key string,
	t time.Time,
) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collections, ok := m.versions[col]
	if !ok {
		return nil, ErrNotFound
	}

	chain, ok := collections[key]
	if !ok {
		return nil, ErrNotFound
	}

	val, exists := chain.asOf(t)
	if !exists {
		return nil, ErrNotFound
	}

	return val, nil
}

// MapExistsAsOf returns true if the key existed at timestamp t.
func (m *memoryEngine) MapExistsAsOf(
	_ context.Context,
	col, key string,
	t time.Time,
) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collections, ok := m.versions[col]
	if !ok {
		return false, nil
	}

	chain, ok := collections[key]
	if !ok {
		return false, nil
	}

	_, exists := chain.asOf(t)

	return exists, nil
}

// Compile-time assertion that memoryEngine implements VersionedStorage.
var _ VersionedStorage = (*memoryEngine)(nil)
