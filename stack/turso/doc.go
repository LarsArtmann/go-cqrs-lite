// Package turso provides a Turso (embedded LibSQL) preset for [stack.Bundle].
//
// It is the recommended preset for deployments that need Turso's embedded
// LibSQL database with optional remote sync. Events, commands, queries,
// snapshots, checkpoints, and read models are all stored in a single Turso
// database file. The event bus uses an in-process GoChannel (LibSQL has no
// pub/sub semantics).
//
// # Quick Start — Local Embedded Database
//
//	b, err := turso.New("app.db")
//	defer b.Close()
//
// # Quick Start — Remote Sync (Offline-First)
//
//	b, err := turso.NewSync(ctx, "local.db",
//	    "libsql://my-db.turso.io", "auth-token")
//	defer b.Close()
//
//	// Local writes work offline. Sync with the remote server:
//	sync := b.Sync()
//	_ = sync.Push(ctx)               // send local writes to remote
//	changed, _ := sync.Pull(ctx)     // receive remote changes
//
// # Production Hardening
//
// Enable CQRS-optimized indexes and performance PRAGMAs:
//
//	b, err := turso.New("app.db", turso.WithOptimizations())
//
// Enable foreign-key enforcement for new databases:
//
//	b, err := turso.New("app.db",
//	    turso.WithOptimizations(),
//	    turso.WithForeignKeys(),
//	)
//
// WAL mode is enabled by default (with synchronous=NORMAL and busy_timeout).
// Disable only if you have a specific reason:
//
//	b, err := turso.New("app.db", turso.WithoutWAL())
//
// # Multi-Database Topology
//
// Split concerns across separate database files (local mode only — sync mode
// requires all stores in one syncing database):
//
//	b, err := turso.New("app.db",
//	    turso.WithEventDB("events.db"),   // events + snapshots + checkpoints
//	    turso.WithQueryDB("queries.db"),  // command + query audit logs
//	    turso.WithViewDB("views.db"),     // materialized views (KV)
//	)
//
// New runs schema migration by default. Disable with [WithoutAutoMigrate].
package turso
