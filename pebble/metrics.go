package pebble

// PebbleMetrics holds key Pebble LSM metrics for operational observability.
type PebbleMetrics struct {
	BlockCacheSize   int64
	BlockCacheHits   int64
	BlockCacheMisses int64
	NumFilesTotal    int64
	NumFilesPerLevel []int64
	CompactionDebt   uint64
	WALFiles         int64
	MemTableSize     uint64
	// CompactionDurationNanos is the total time spent compacting.
	CompactionDurationNanos int64
}

// Metrics returns key operational metrics from the underlying Pebble database.
// Use these for health checks, capacity planning, and performance debugging.
func (b *Backend) Metrics() PebbleMetrics {
	m := b.database.Metrics()

	numFiles := int64(0)
	perLevel := make([]int64, len(m.Levels))

	for i, lvl := range m.Levels {
		perLevel[i] = int64(lvl.NumFiles)
		numFiles += int64(lvl.NumFiles)
	}

	return PebbleMetrics{
		BlockCacheSize:          m.BlockCache.Size,
		BlockCacheHits:          m.BlockCache.Hits,
		BlockCacheMisses:        m.BlockCache.Misses,
		NumFilesTotal:           numFiles,
		NumFilesPerLevel:        perLevel,
		CompactionDebt:          m.Compact.EstimatedDebt,
		WALFiles:                m.WAL.Files,
		MemTableSize:            m.MemTable.Size,
		CompactionDurationNanos: int64(m.Compact.Duration),
	}
}
