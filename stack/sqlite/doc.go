// Package sqlite provides a SQLite-backed preset for [stack.Bundle].
//
// It is the recommended preset for single-node deployments that need
// persistence: events, commands, queries, snapshots, checkpoints, and read
// models are all stored in a single SQLite database file. The event bus uses
// an in-process GoChannel (SQLite has no pub/sub semantics).
//
// # Quick Start
//
//	b, err := sqlite.New("log.db")
//	defer b.Close()
//
// # Production Hardening
//
// Enable foreign-key enforcement for new databases where referential integrity
// is required:
//
//	b, err := sqlite.New("log.db", sqlite.WithForeignKeys())
//
// WAL mode (with busy_timeout=5000) is enabled by default. Disable only if you
// have a specific reason (e.g. a network filesystem that doesn't support WAL):
//
//	b, err := sqlite.New("log.db", sqlite.WithoutWAL())
//
// Apply CQRS-optimized PRAGMAs (cache_size, temp_store, mmap_size) for
// production throughput:
//
//	b, err := sqlite.New("log.db", sqlite.WithOptimizations())
//
// # Filesystem Considerations
//
// WAL mode relies on mmap and file locking that behave differently across
// filesystems. The default (WAL enabled) is correct for local filesystems.
//
//	| Filesystem | WAL support | Notes                                                  |
//	| ---------- | ----------- | ------------------------------------------------------ |
//	| ext4/xfs   | Full        | No special action.                                     |
//	| ZFS        | Full        | Honors fsync correctly; one of the safest choices.     |
//	| btrfs      | Works       | CoW fragments the WAL under append-heavy load. Set     |
//	|            |             | NOCOW (chattr +C) on the DB directory before creating  |
//	|            |             | the database to avoid fragmentation. Tradeoff: that    |
//	|            |             | directory is excluded from btrfs snapshots.            |
//	| NFS/SMB    | Broken      | mmap + locking unreliable. Use [WithoutWAL].           |
//
// NOCOW only takes effect on an empty file or directory, so set it before
// creating the database:
//
//	mkdir /var/lib/myapp && chattr +C /var/lib/myapp
//	b, err := sqlite.New("/var/lib/myapp/log.db")
//
// # Multi-Database Topology
//
// Split concerns across separate database files to eliminate reader/writer
// contention in production:
//
//	b, err := sqlite.New("primary.db",
//	    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
//	    sqlite.WithQueryDB("queries.db"),  // command + query audit logs
//	    sqlite.WithViewDB("views.db"),     // materialized views (KV)
//	)
//
// | Database    | Contains                       | Rationale                                          |
// | ----------- | ------------------------------ | -------------------------------------------------- |
// | **Event DB**  | events, snapshots, checkpoints | Event-sourcing write model, isolated from reads   |
// | **Query DB**  | commands, queries              | Operational audit log, isolated from write model  |
// | **View DB**   | materialized views (cqrs_kv)   | Read scans don't contend with event appends       |
//
// Default (single database) is fine for development and low-traffic apps.
//
// New enables WAL mode and runs schema migration by default. Disable either
// with [WithoutWAL] or [WithoutAutoMigrate].
package sqlite
