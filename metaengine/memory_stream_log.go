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

func (m *memoryEngine) JournalReadFrom(
	_ context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
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
// Individual operations (Map, Counter, etc.) already hold the mutex
// independently, so for cross-collection Store.InTransaction this provides
// sequential execution. For optimistic concurrency on stream logs, engines
// should implement AtomicAppender instead — see [StreamAppendExpected].
func (m *memoryEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// StreamAppendExpected appends values only if the stream's current version
// matches expectedVersion. This is the atomic optimistic-concurrency
// primitive: the version check and append happen under a single lock
// acquisition, eliminating the race between check and append.
//
// Returns event.ErrVersionConflict if the current version doesn't match.
func (m *memoryEngine) StreamAppendExpected(
	_ context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := int64(len(m.data.streams[col][sid]))
	if current != expectedVersion {
		return ErrVersionConflict
	}

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
