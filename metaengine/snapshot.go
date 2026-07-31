package metaengine

import (
	"context"
	"fmt"
)

// SnapshotEntry is a single key-value pair in a collection snapshot.
type SnapshotEntry struct {
	Key   string
	Value []byte // raw JSON-encoded value
}

// CollectionSnapshot holds all entries in a single collection.
type CollectionSnapshot struct {
	Name    string
	Engine  string
	Entries []SnapshotEntry
}

// Snapshot exports the current state of all Map-type collections. This is
// suitable for backup, transfer between engines, or debugging.
//
// Collections backed by non-Map ADTs (Counter, Set, Graph) are skipped —
// they derive their state from the event stream and can be rebuilt by
// replaying events.
//
//	snap, _ := store.Snapshot(ctx)
//	// snap[0].Name = "tasks"
//	// snap[0].Entries = [{Key: "task-1", Value: [...]}, ...]
func (s *Store) Snapshot(ctx context.Context) ([]CollectionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CollectionSnapshot, 0, len(s.queries))

	for name, q := range s.queries {
		// Only Map ADTs have exportable key-value entries.
		if q.adt != ADTMap {
			continue
		}

		sb, ok := q.engine.(ScanBackend)
		if !ok {
			continue
		}

		rows, err := sb.MapScan(ctx, name, nil, nil, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("snapshot %s: %w", name, err)
		}

		entries := make([]SnapshotEntry, 0, len(rows))

		for _, row := range rows {
			entries = append(entries, SnapshotEntry{
				Key:   row.Key,
				Value: row.Value,
			})
		}

		result = append(result, CollectionSnapshot{
			Name:    name,
			Engine:  q.engine.Profile().Name,
			Entries: entries,
		})
	}

	return result, nil
}
