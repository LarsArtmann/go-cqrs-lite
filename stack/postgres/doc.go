// Package postgres provides a PostgreSQL-backed preset for [stack.Bundle].
//
// It wires events, commands, queries, snapshots, and checkpoints to a single
// PostgreSQL database via the storage.SQLBackend facade. By default the event
// bus is in-memory (memory.NewMemoryBus) for single-process use. For
// multi-process pub/sub, pass [WithDistributedBus] with a [PgxListener] to
// wire storage.PostgresBus (Postgres LISTEN/NOTIFY). Read models persist to
// the database via storage.SQLKVStore.
//
// Single-process (default):
//
//	b, err := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
//	defer b.Close()
//
// Multi-process (LISTEN/NOTIFY for cross-process event propagation):
//
//	listener, _ := postgres.NewPgxListenerFromDSN(ctx, dsn)
//	b, err := postgres.New(dsn, postgres.WithDistributedBus(listener))
//	defer b.Close()
//
// All data is persistent. The returned Bundle owns the *sql.DB; Close releases it.
// When WithDistributedBus is used, Close also drains the listener's dedicated
// LISTEN connection.
package postgres
