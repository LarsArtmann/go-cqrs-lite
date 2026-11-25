package metaengine

import (
	"maps"
	"slices"
)

// Plan rules run without the store lock (layout rules write DDL), so anything
// they read from the Store must be copied under the read lock: AddEngine and
// RegisterQuery mutate these concurrently with a background Replan.

func (s *Store) enginesForRules() []Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.engines)
}

func (s *Store) queriesForRules() map[string]queryMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return maps.Clone(s.queries)
}
