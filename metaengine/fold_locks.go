package metaengine

import "sync"

// foldLocks hands out one mutex per query name (METAENGINE-LAYOUT-ROLES.md §7).
//
// Fold instances are owned by exactly one query, and the mutable state shared
// between fold invocations (RecordAwareFold.SetCurrentRecord) lives on the fold
// instance. Locking per query name therefore serializes exactly the fold
// invocations that share state, while folds belonging to different queries
// apply in parallel — including across the primary dispatch path and the
// replication applier, which share fold instances.
//
// Lock ordering: query locks are acquired inside the store read lock (primary
// dispatch) or after releasing it (replication applies against a snapshot).
// The store write lock never wraps a query lock, so no deadlock cycle exists.
type foldLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newFoldLocks() *foldLocks {
	return &foldLocks{locks: make(map[string]*sync.Mutex)}
}

// get returns the mutex guarding the named query's fold state. The registry
// mutex is held only for the lookup/create itself, never during fold work.
func (f *foldLocks) get(queryName string) *sync.Mutex {
	f.mu.Lock()
	defer f.mu.Unlock()

	if l, ok := f.locks[queryName]; ok {
		return l
	}

	l := &sync.Mutex{}
	f.locks[queryName] = l

	return l
}
