package sqlopt

// DSNConfig holds the multi-database override strings shared by every SQL
// preset. Each preset embeds DSNConfig in its private config struct; the
// shared [DSNConfig.WithoutAutoMigrate], [DSNConfig.SetEventDB],
// [DSNConfig.SetQueryDB], and [DSNConfig.SetViewDB] methods eliminate the
// per-preset option-function duplication.
//
// Turso uses path-based overrides (eventPath vs eventDSN) but the semantics
// are identical; turso adapts via a thin wrapper that reads its own fields.
type DSNConfig struct {
	AutoMigrate bool
	EventDSN    string
	QueryDSN    string
	ViewDSN     string
}

// WithoutAutoMigrate disables schema creation. Use this when you manage
// schemas yourself (e.g. via a migration tool). By default New creates all
// required tables.
func (c *DSNConfig) WithoutAutoMigrate() {
	c.AutoMigrate = false
}

// SetEventDB sets a separate DSN for the event store. When set, events,
// snapshots, and checkpoints are persisted to this database instead of the
// primary DSN. The deployer chooses this when isolating write-heavy event
// streams from query traffic.
func (c *DSNConfig) SetEventDB(dsn string) {
	c.EventDSN = dsn
}

// SetQueryDB sets a separate DSN for the command and query audit stores.
// When set, persisted commands and queries go to this database.
func (c *DSNConfig) SetQueryDB(dsn string) {
	c.QueryDSN = dsn
}

// SetViewDB sets a separate DSN for the read-model KV store. When set,
// materialized views are persisted to this database, isolating read-model
// scans from the event store.
func (c *DSNConfig) SetViewDB(dsn string) {
	c.ViewDSN = dsn
}
