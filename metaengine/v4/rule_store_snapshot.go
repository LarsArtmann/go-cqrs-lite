package metaengine

import "slices"

// Plan rules run without the store lock (layout rules write DDL), so anything
// they read from the Store must be copied under the read lock: AddEngine and
// RegisterQuery mutate these, and Replan re-assigns queries, concurrently with
// a background Replan's rules.

func (s *Store) enginesForRules() []Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.engines)
}

func (s *Store) queriesForRules() map[string]queryMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]queryMeta, len(s.queries))
	for name, meta := range s.queries {
		out[name] = s.snapshotForRule(meta)
	}

	return out
}
