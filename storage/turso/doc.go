// Package turso provides CQRS storage adapters for Turso databases (embedded
// LibSQL with optional remote sync). It is the recommended integration path
// for applications that want SQLite-compatible storage with offline-first
// sync, edge deployment, or Turso's serverless data platform.
//
// Turso uses the same SQL dialect as SQLite, so this package delegates store
// construction to [github.com/larsartmann/go-cqrs-lite/storage/v3] with the
// SQLite dialect. The value-add here is:
//
//   - Phantom-typed inputs ([DbPath], [RemoteURL], [AuthToken]) for compile-time safety.
//   - A [Backend] facade exposing all five CQRS stores via one *sql.DB.
//   - Remote sync via [OpenSync] / [OpenSyncWithConfig] with [SyncOption] tuning.
//   - Production helpers: [ConfigurePool], [InitSchemaWithIndexesAndOptimizations].
//   - The [turso/indexing] sub-package for auto-smart index management.
//
// # Quick Start — Local Embedded Database
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	defer db.Close()
//	turso.ConfigurePool(db)
//
//	_ = turso.InitSchema(ctx, db)
//	store, _ := turso.NewEventStore(db)
//
// # Quick Start — Full Stack via Backend Facade
//
// The [Backend] facade exposes event, command, query, snapshot, and checkpoint
// stores sharing a single connection:
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	defer db.Close()
//	turso.ConfigurePool(db)
//
//	backend, _ := turso.NewBackend(db)
//	defer backend.Close()
//
//	eventStore := backend.EventStore()           // eager
//	cmdStore, _ := backend.CommandStore()         // lazy, goroutine-safe
//	qStore, _   := backend.QueryStore()           // lazy, goroutine-safe
//	snapStore, _ := backend.SnapshotStore()        // lazy, goroutine-safe
//	cpStore, _  := backend.CheckpointStore()       // lazy, goroutine-safe
//
// # Quick Start — Remote Sync (Offline-First)
//
//	syncDB, _ := turso.OpenSync(ctx,
//	    turso.DbPath("local.db"),
//	    turso.RemoteURL("libsql://my-db.turso.io"),
//	    turso.AuthToken("token"),
//	)
//	defer syncDB.Close()
//
//	_ = syncDB.Push(ctx)                // send local writes to remote
//	changed, _ := syncDB.Pull(ctx)      // receive remote changes
//	_ = syncDB.HealthCheck(ctx)         // verify connection for health probes
//	stats, _ := syncDB.Stats(ctx)       // WAL size, bytes sent/received
//
// For advanced sync configuration (client name, long-poll timeout, busy
// timeout, namespace, bootstrap behavior):
//
//	syncDB, _ := turso.OpenSyncWithConfig(ctx, path, url, token,
//	    turso.WithSyncClientName("my-app"),
//	    turso.WithSyncLongPollTimeout(30*time.Second),
//	    turso.WithSyncBusyTimeout(10*time.Second),
//	    turso.WithSyncBootstrapIfEmpty(false),
//	)
//
// # Production Setup — Schema + Indexes + Pragmas
//
// For new databases, use the one-shot initializer to create all tables,
// apply CQRS-optimized indexes, and set performance PRAGMAs:
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	turso.ConfigurePool(db)
//	_ = turso.InitSchemaWithIndexesAndOptimizations(ctx, db)
//
// # Connection Pool
//
// Embedded LibSQL serializes writes through a single connection. Always call
// [ConfigurePool] after opening to cap MaxOpenConns at 1, preventing
// "database is locked" errors under concurrent load:
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	turso.ConfigurePool(db)
//
// # Indexing Sub-Package
//
// See [github.com/larsartmann/go-cqrs-lite/storage/turso/v3/indexing] for the
// auto-smart index advisor, auto-indexer, usage statistics, and WAL
// checkpoint scheduler.
package turso
