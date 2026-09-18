package metaengine

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// versionedEntry records a single value at a point in time.
type versionedEntry struct {
	ts    time.Time
	value any // nil means the key was deleted at this timestamp (tombstone)
}

// versionChain stores the append-only history of a single key, ordered by
// timestamp ascending. Writes may arrive out of order (replays, clock skew),
// so entries are inserted at their sorted position; binary search finds the
// latest entry <= t. A same-timestamp write lands AFTER existing entries —
// last-writer-wins, mirroring BigTable cell dedup (ADR-0141 §1).
type versionChain struct {
	entries []versionedEntry
}

// asOf returns the value and existence at timestamp t. The value is the
// latest entry with ts <= t. If that entry is a tombstone, the key was
// deleted at that time and asOf reports (nil, false).
func (vc *versionChain) asOf(t time.Time) (any, bool) {
	idx := sort.Search(len(vc.entries), func(i int) bool {
		return vc.entries[i].ts.After(t)
	})

	if idx == 0 {
		return nil, false // no entries at or before t
	}

	entry := vc.entries[idx-1] // latest entry <= t

	if entry.value == nil {
		return nil, false // tombstoned before t
	}

	return entry.value, true
}

// history returns the surviving versions in [from, to], newest-first.
func (vc *versionChain) history(from, to time.Time) []CellVersion {
	idx := sort.Search(len(vc.entries), func(i int) bool {
		return vc.entries[i].ts.After(to)
	})

	var out []CellVersion

	for i := idx - 1; i >= 0 && !vc.entries[i].ts.Before(from); i-- {
		out = append(out, CellVersion{Timestamp: vc.entries[i].ts, Value: vc.entries[i].value})
	}

	return out
}

// insertAt places entry at its sorted position, after any same-timestamp
// entries (last-writer-wins).
func (vc *versionChain) insertAt(entry versionedEntry) {
	idx := sort.Search(len(vc.entries), func(i int) bool {
		return vc.entries[i].ts.After(entry.ts)
	})

	vc.entries = append(vc.entries, versionedEntry{})
	copy(vc.entries[idx+1:], vc.entries[idx:])
	vc.entries[idx] = entry
}

// chainLocked returns the version chain for (col, key), creating it when
// absent. Caller MUST hold m.mu.Lock().
func (m *memoryEngine) chainLocked(col, key string) *versionChain {
	if m.versions[col] == nil {
		m.versions[col] = make(map[string]*versionChain)
	}

	chain, ok := m.versions[col][key]
	if !ok {
		chain = &versionChain{}
		m.versions[col][key] = chain
	}

	return chain
}

// recordVersionAt records a timestamped entry on the key's chain (sorted
// insert + retention trim), without touching the latest view — the caller
// has already written the main map. Caller MUST hold m.mu.Lock() and MUST
// have versioning enabled (m.versions != nil).
func (m *memoryEngine) recordVersionAt(col, key string, value any, ts time.Time) {
	chain := m.chainLocked(col, key)
	chain.insertAt(versionedEntry{ts: ts, value: value})
	m.trimRetentionLocked(chain, ts)
}

// applyVersionLocked records one timestamped version and syncs the latest
// view: the main map always mirrors the chain's NEWEST entry, so out-of-order
// writes cannot regress MapGet. A nil value is a tombstone. The chain is
// keyed by the string form of key; the main map keeps the NATIVE key so
// latest reads stay type-faithful. Caller MUST hold m.mu.Lock(). When
// versioning is disabled this degrades to a plain set/delete (latest-only,
// zero history overhead).
func (m *memoryEngine) applyVersionLocked(col string, key any, value any, ts time.Time) {
	if m.versions == nil {
		store := m.getMapLocked(col)
		if value == nil {
			delete(store, key)
		} else {
			store[key] = value
		}

		return
	}

	chain := m.chainLocked(col, fmt.Sprint(key))
	chain.insertAt(versionedEntry{ts: ts, value: value})
	m.trimRetentionLocked(chain, ts)
	m.syncLatestLocked(col, key, chain)
}

// trimRetentionLocked prunes the chain per the configured RetentionPolicy
// (MaxVersions / MaxAge), never removing the newest entry. Caller MUST hold
// m.mu.Lock().
func (m *memoryEngine) trimRetentionLocked(chain *versionChain, newest time.Time) {
	if m.retention == nil {
		return
	}

	if m.retention.MaxVersions > 0 && len(chain.entries) > m.retention.MaxVersions {
		cutoff := len(chain.entries) - m.retention.MaxVersions
		chain.entries = chain.entries[cutoff:]
	}

	if m.retention.MaxAge > 0 {
		minTs := newest.Add(-m.retention.MaxAge)
		idx := sort.Search(len(chain.entries), func(i int) bool {
			return chain.entries[i].ts.After(minTs)
		})

		switch {
		case idx == 0:
			// Everything within the window: keep all.
		case idx < len(chain.entries):
			chain.entries = chain.entries[idx:]
		default:
			// Everything older than the window: keep only the newest entry
			// (a live cell never vanishes from latest reads).
			chain.entries = chain.entries[len(chain.entries)-1:]
		}
	}
}

// syncLatestLocked mirrors the chain's newest entry into the main map so
// MapGet/MapScan stay O(1) latest reads. Caller MUST hold m.mu.Lock().
func (m *memoryEngine) syncLatestLocked(col string, key any, chain *versionChain) {
	store := m.getMapLocked(col)

	tail := chain.entries[len(chain.entries)-1]
	if tail.value == nil {
		delete(store, key)
	} else {
		store[key] = tail.value
	}
}

// CellVersioningEnabled implements [CellVersioningToggle]: the memory engine
// records cell versions only when constructed via
// NewMemoryEngineWithVersioning. Wrappers that embed *memoryEngine inherit
// the toggle, so the Store's fold path keeps using their MapSet/MapUpdate
// overrides when versioning is off.
func (m *memoryEngine) CellVersioningEnabled() bool { return m.versions != nil }

// --- VersionedStorage implementation ---

// MapGetAsOf returns the value for a key as it existed at timestamp t.
// Returns ErrNotFound if the key did not exist at that time.
func (m *memoryEngine) MapGetAsOf(
	_ context.Context,
	col, key string,
	t time.Time,
) (any, error) {
	//art-dupl:accept guard prologue twin of MapExistsAsOf below — lock+chainFor with divergent return types
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chainFor(col, key)
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

	chain, ok := m.chainFor(col, key)
	if !ok {
		return false, nil
	}

	_, exists := chain.asOf(t)

	return exists, nil
}

func (m *memoryEngine) chainFor(col, key string) (*versionChain, bool) {
	if m.versions == nil {
		return nil, false
	}

	collections, ok := m.versions[col]
	if !ok {
		return nil, false
	}

	chain, ok := collections[key]

	return chain, ok
}

// --- VersionedWriter implementation ---

// MapSetAt writes value as the (collection, key) version at ts (ADR-0141 §1).
// When versioning is disabled the write degrades to a plain latest-only set.
func (m *memoryEngine) MapSetAt(
	_ context.Context,
	col string,
	key any,
	value any,
	ts time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.applyVersionLocked(col, key, value, ts)

	return nil
}

// MapDeleteAt records a tombstone for (collection, key) at ts. When
// versioning is disabled the call degrades to a plain latest-only delete.
func (m *memoryEngine) MapDeleteAt(
	_ context.Context,
	col string,
	key any,
	ts time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.applyVersionLocked(col, key, nil, ts)

	return nil
}

// --- VersionedUpdater implementation ---

// MapUpdateAt atomically applies update to the current latest value and
// records the result as the version at ts (single lock acquisition).
func (m *memoryEngine) MapUpdateAt(
	_ context.Context,
	col string,
	key any,
	update func(prev any) any,
	ts time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.getMapLocked(col)
	newVal := update(store[key])
	store[key] = newVal

	if m.versions == nil { // opt-in versioning disabled: latest-only
		return nil
	}

	m.recordVersionAt(col, fmt.Sprint(key), newVal, ts)

	return nil
}

// --- CellHistoryReader implementation ---

// MapHistory returns the surviving versions of (collection, key) within
// [from, to], newest-first, tombstones included (nil Value).
func (m *memoryEngine) MapHistory(
	_ context.Context,
	col, key string,
	from, to time.Time,
) ([]CellVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chainFor(col, key)
	if !ok {
		return nil, nil
	}

	return chain.history(from, to), nil
}

// Compile-time assertions that memoryEngine implements the temporal
// capability set (ADR-0141): reads, writes, and history.
var (
	_ VersionedStorage  = (*memoryEngine)(nil)
	_ VersionedWriter   = (*memoryEngine)(nil)
	_ VersionedUpdater  = (*memoryEngine)(nil)
	_ CellHistoryReader = (*memoryEngine)(nil)
)
