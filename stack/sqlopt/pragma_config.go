package sqlopt

// PragmaConfig holds SQLite PRAGMA settings shared by SQLite-based presets
// (sqlite, turso). Each preset embeds PragmaConfig in its private config
// struct and exposes the shared options via a WithPragmas adapter.
type PragmaConfig struct {
	WAL         bool
	Optimize    bool
	ForeignKeys bool
}

// PragmaOption is a functional option for [PragmaConfig].
type PragmaOption func(*PragmaConfig)

// WithoutWAL disables WAL mode. By default presets enable WAL plus a busy
// timeout to eliminate "database is locked" errors under concurrency.
func WithoutWAL() PragmaOption {
	return func(c *PragmaConfig) { c.WAL = false }
}

// WithOptimizations enables CQRS-optimized indexes and performance PRAGMAs
// after schema creation. Recommended for production.
func WithOptimizations() PragmaOption {
	return func(c *PragmaConfig) { c.Optimize = true }
}

// WithForeignKeys enables foreign-key enforcement. Off by default because
// existing databases may contain orphaned references.
func WithForeignKeys() PragmaOption {
	return func(c *PragmaConfig) { c.ForeignKeys = true }
}
