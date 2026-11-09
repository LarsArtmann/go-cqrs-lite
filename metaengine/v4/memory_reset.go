package metaengine

import "context"

// newMemData returns an empty memData with every ADT collection map initialized
// (graphs stay nil and are created lazily). Shared by NewMemoryEngine and
// ResetEngine so a reset returns the engine to its exact post-construction state.
func newMemData() *memData {
	return &memData{
		maps:          make(map[string]map[any]any),
		sets:          make(map[string]map[any]struct{}),
		counters:      make(map[string]map[string]int64),
		multimaps:     make(map[string]map[any][]any),
		logs:          make(map[string][]any),
		streams:       make(map[string]map[string][]any),
		streamJournal: make(map[string][]streamJournalEntry),
	}
}

// ResetEngine implements [EngineResetter]: it drops ALL materialized state —
// every derived ADT collection, the version chains (when versioning is
// enabled), and the vector/search/spatial indexes. The JOURNAL (logs,
// streams, streamJournal) survives: journal entries are facts on the
// ADR-0136 invertibility ladder (ADR-0143) — the replay source a reset
// rebuilds FROM, never derived data a reset clears.
func (m *memoryEngine) ResetEngine(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	journal := m.data.logs
	streams := m.data.streams
	streamJournal := m.data.streamJournal

	m.data = newMemData()
	m.data.logs = journal
	m.data.streams = streams
	m.data.streamJournal = streamJournal

	m.vectorIdx = NewMemoryVectorIndex()
	m.searchIdx = NewMemorySearchIndex()
	m.spatialIdx = NewMemorySpatialIndex()

	if m.versions != nil {
		m.versions = make(map[string]map[string]*versionChain)
	}

	return nil
}
