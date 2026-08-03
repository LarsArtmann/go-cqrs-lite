// Package postgres provides a PostgreSQL-backed preset for [stack.Bundle].
//
// It wires events, commands, queries, snapshots, and checkpoints to a single
// PostgreSQL database via the storage.SQLBackend facade. The event bus is
// watermill.EventBus (GoChannel) for single-process use. Read models persist to
// the database via storage.SQLKVStore.
//
// Single-process (default):
//
//	b, err := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
//	defer b.Close()
//
// Multi-DB split (separate databases for events, audit, and views):
//
//	b, err := postgres.New(primaryDSN,
//	    postgres.WithDSN(
//	        sqlopt.WithEventDB("postgres://host/events_db"),
//	        sqlopt.WithQueryDB("postgres://host/queries_db"),
//	        sqlopt.WithViewDB("postgres://host/views_db"),
//	    ),
//	)
//	defer b.Close()
//	// Events+snapshots+checkpoints → events_db
//	// Commands+queries → queries_db
//	// Read models → views_db
//
// All data is persistent. The returned Bundle owns the *sql.DB; Close releases it.
package postgres
