// Package sqlite provides a SQLite-backed preset for [stack.Bundle].
//
// It is the recommended preset for single-node deployments that need
// persistence: events, commands, queries, snapshots, and checkpoints are all
// stored in a single SQLite database file. The event bus and read-model
// backend use in-memory implementations (the bus has no persistence
// semantics; a SQL-backed KV store for read models is future work).
//
//	b, err := sqlite.New("log.db")
//	defer b.Close()
//
// For a split topology (separate databases for the append log and the view
// store), use [AppendLog] and [Views] as [stack.Option] values with
// [stack.New]:
//
//	b, err := stack.New(
//	    sqlite.AppendLog("log.db"),
//	    sqlite.Views("views.db"),
//	)
//
// New enables WAL mode and runs schema migration by default. Disable either
// with [WithoutWAL] or [WithoutAutoMigrate].
package sqlite
