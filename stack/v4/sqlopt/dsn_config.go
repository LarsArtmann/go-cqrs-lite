package sqlopt

// ApplyTo applies a slice of functional options to a config target.
// Used by preset adapters (WithDSN, WithPragmas) to keep them concise.
func ApplyTo[C any, O ~func(*C)](opts []O, c *C) {
	for _, opt := range opts {
		opt(c)
	}
}

// DSNConfig holds the multi-database override strings shared by every SQL
// preset. Each preset embeds DSNConfig in its private config struct and exposes
// the shared DSNOption functions via a WithDSN adapter (e.g.
// sqlite.WithDSN, postgres.WithDSN, turso.WithDSN).
//
// Turso uses path-based overrides (eventPath vs eventDSN) but the semantics
// are identical; turso adapts via a thin wrapper that reads its own fields.
type DSNConfig struct {
	AutoMigrate bool
	EventDSN    string
	QueryDSN    string
	ViewDSN     string
}

// DSNOption is a functional option for [DSNConfig]. Each SQL preset provides a
// [WithDSN] adapter (e.g. sqlite.WithDSN, postgres.WithDSN, turso.WithDSN)
// that applies these shared options to its embedded DSNConfig, eliminating
// per-preset option-function duplication:
//
//	sqlite.New(dsn, sqlite.WithDSN(
//	    sqlopt.WithEventDB("events.db"),
//	    sqlopt.WithViewDB("views.db"),
//	))
type DSNOption func(*DSNConfig)

// WithoutAutoMigrate disables schema creation. Use this when you manage
// schemas yourself (e.g. via a migration tool). By default New creates all
// required tables.
func WithoutAutoMigrate() DSNOption {
	return func(c *DSNConfig) { c.AutoMigrate = false }
}

// WithEventDB sets a separate DSN for the event store. When set, events,
// snapshots, and checkpoints are persisted to this database instead of the
// primary DSN. The deployer chooses this when isolating write-heavy event
// streams from query traffic.
func WithEventDB(dsn string) DSNOption {
	return func(c *DSNConfig) { c.EventDSN = dsn }
}

// WithQueryDB sets a separate DSN for the command and query audit stores.
// When set, persisted commands and queries go to this database.
func WithQueryDB(dsn string) DSNOption {
	return func(c *DSNConfig) { c.QueryDSN = dsn }
}

// WithViewDB sets a separate DSN for the read-model KV store. When set,
// materialized views are persisted to this database, isolating read-model
// scans from the event store.
func WithViewDB(dsn string) DSNOption {
	return func(c *DSNConfig) { c.ViewDSN = dsn }
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
