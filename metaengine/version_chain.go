package metaengine

// The version-chain data structure behind temporal Memory engines
// (ADR-0141): an append-only, timestamp-sorted history per key with
// as-of resolution, range history, and last-writer-wins same-ts insert.

import (
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
