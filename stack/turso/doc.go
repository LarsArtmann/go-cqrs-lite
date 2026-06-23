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
// # Multi-Database Topology
//
// For split topologies (separate databases for the append log and the view
// store), use [WithEventDB], [WithQueryDB], and [WithViewDB]:
//
//	b, err := turso.New("app.db",
//	    turso.WithEventDB("events.db"),
//	    turso.WithViewDB("views.db"),
//	)
//
// New runs schema migration by default. Disable with [WithoutAutoMigrate].
package turso
