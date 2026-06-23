// Package sqlite provides a SQLite-backed preset for [stack.Bundle].
//
// It is the recommended preset for single-node deployments that need
// persistence: events, commands, queries, snapshots, checkpoints, and read
// models are all stored in a single SQLite database file. The event bus uses
// an in-process GoChannel (SQLite has no pub/sub semantics).
//
//	b, err := sqlite.New("log.db")
//	defer b.Close()
//
// For a split topology (separate databases for events, audit logs, and
// materialized views), use the multi-DB options:
//
//	b, err := sqlite.New("primary.db",
//	    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
//	    sqlite.WithQueryDB("queries.db"),  // command + query audit logs
//	    sqlite.WithViewDB("views.db"),     // materialized views (KV)
//	)
//
// New enables WAL mode and runs schema migration by default. Disable either
// with [WithoutWAL] or [WithoutAutoMigrate].
package sqlite
