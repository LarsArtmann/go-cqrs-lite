package metaengine

// task_snapshot.go holds the immutable (query, fold) task index shared by the
// primary dispatch path and the replication appliers (METAENGINE-LAYOUT-ROLES.md).
//
// The snapshot exists because PromoteEngine holds the store write lock while a
// shadow engine drains its replication backlog. The applier goroutine must
// therefore resolve fold tasks WITHOUT taking the store read lock (that would
// deadlock). The map is rebuilt under the write lock whenever queries change
// (Plan, RegisterQuery) and swapped atomically; readers never lock.

// rebuildTaskSnapLocked recomputes the event-type → fold-task index from the
// registered queries. The caller must hold the store write lock (or be
// single-threaded during Plan). One fold per query per event type — selected
// via QueryFoldByEvent, exactly matching dispatchFolds semantics.
func (s *Store) rebuildTaskSnapLocked() {
	snap := make(map[string][]foldTask)

	for _, name := range sortedQueryNames(s.queries) {
		q := s.queries[name]

		for eventType, idx := range q.QueryFoldByEvent() {
			snap[eventType] = append(snap[eventType], foldTask{q: q, fold: q.QueryFolds()[idx]})
		}
	}

	s.taskSnap.Store(&snap)
}

// tasksFor returns the fold tasks matching an event type, in deterministic
// query order. Lock-free; returns nil when no query listens to the event.
func (s *Store) tasksFor(eventType string) []foldTask {
	if p := s.taskSnap.Load(); p != nil {
		return (*p)[eventType]
	}

	return nil
}

// filterTasks applies an optional query filter to a task slice.
func filterTasks(tasks []foldTask, queryFilter map[string]bool) []foldTask {
	if queryFilter == nil {
		return tasks
	}

	out := make([]foldTask, 0, len(tasks))

	for _, t := range tasks {
		if queryFilter[t.q.QueryName()] {
			out = append(out, t)
		}
	}

	return out
}
