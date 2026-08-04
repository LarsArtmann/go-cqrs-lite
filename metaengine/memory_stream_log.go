package metaengine

import (
	"context"
	"slices"
)

// --- StreamLogBackend ---

func (m *memoryEngine) StreamAppend(_ context.Context, col, sid string, values []any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data.streams[col] == nil {
		m.data.streams[col] = make(map[string][]any)
	}

	m.data.streams[col][sid] = append(m.data.streams[col][sid], values...)

	for _, v := range values {
		nextSeq := int64(1)
		if j := m.data.streamJournal[col]; len(j) > 0 {
			nextSeq = j[len(j)-1].seq + 1
		}

		m.data.streamJournal[col] = append(m.data.streamJournal[col], streamJournalEntry{
			seq:      nextSeq,
			streamID: sid,
			value:    v,
		})
	}

	return nil
}

func (m *memoryEngine) StreamRead(_ context.Context, col, sid string) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return slices.Clone(m.data.streams[col][sid]), nil
}

func (m *memoryEngine) StreamVersion(_ context.Context, col, sid string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return int64(len(m.data.streams[col][sid])), nil
}

func (m *memoryEngine) JournalReadAll(_ context.Context, col string) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	journal := m.data.streamJournal[col]
	result := make([]any, 0, len(journal))

	for _, entry := range journal {
		result = append(result, entry.value)
	}

	return result, nil
}

func (m *memoryEngine) JournalReadFrom(_ context.Context, col string, afterSeq int64, limit int) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	journal := m.data.streamJournal[col]

	start := 0
	for start < len(journal) && journal[start].seq <= afterSeq {
		start++
	}

	remaining := journal[start:]
	if limit <= 0 || limit > len(remaining) {
		limit = len(remaining)
	}

	result := make([]any, 0, limit)
	for i := range limit {
		result = append(result, remaining[i].value)
	}

	return result, nil
}

// RunInTx executes fn without additional locking for the memory engine.
// Individual operations already hold the mutex, so for the Memory engine
// this provides best-effort atomicity (sufficient for tests; SQLite provides
// true transactional isolation).
func (m *memoryEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
