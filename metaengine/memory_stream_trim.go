package metaengine

import (
	"context"
	"slices"
)

// TrimStream deletes one stream's values and its journal entries and returns
// the number of journal rows removed. Implements StreamTrimmer for the memory
// engine: the in-process maps hold no bytes to erase, but the contract
// (trimmed stream folds to nothing, next append is a fresh stream) is the
// same, and tests ride it. The memory engine keeps no snapshots, so there is
// nothing to co-trim.
func (m *memoryEngine) TrimStream(_ context.Context, col, sid string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := int64(len(m.data.streams[col][sid]))
	delete(m.data.streams[col], sid)

	m.data.streamJournal[col] = slices.DeleteFunc(m.data.streamJournal[col],
		func(entry streamJournalEntry) bool {
			return entry.streamID == sid
		})

	return removed, nil
}
