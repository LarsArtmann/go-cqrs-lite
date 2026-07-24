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

// BlockCacheHitRate returns the block cache hit rate as a fraction [0.0, 1.0].
// Returns 0.0 when there are no cache accesses yet.
func (m PebbleMetrics) BlockCacheHitRate() float64 {
	total := m.BlockCacheHits + m.BlockCacheMisses
	if total == 0 {
		return 0
	}

	return float64(m.BlockCacheHits) / float64(total)
}

// Metrics returns key operational metrics from the underlying Pebble database.
// Use these for health checks, capacity planning, and performance debugging.
func (b *Backend) Metrics() PebbleMetrics {
	m := b.database.Metrics()

	numFiles := int64(0)
	perLevel := make([]int64, len(m.Levels))

	for i, lvl := range m.Levels {
		perLevel[i] = lvl.NumFiles
		numFiles += lvl.NumFiles
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

// DiskUsage returns the total disk space used by the Pebble database in bytes,
// including live SSTables, WAL files, and obsolete files not yet deleted.
// This is more precise than a filesystem walk because it is computed from
// Pebble's internal metrics rather than scanning the directory.
func (b *Backend) DiskUsage() uint64 {
	m := b.database.Metrics()

	var total uint64

	for _, lvl := range m.Levels {
		total += uint64(lvl.Size)
	}

	total += m.WAL.PhysicalSize
	total += m.WAL.ObsoletePhysicalSize
	total += m.Table.ObsoleteSize
	total += m.Table.ZombieSize

	return total
}
