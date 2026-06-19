// Package postgres provides a PostgreSQL-backed preset for [stack.Bundle].
//
// It wires events, commands, queries, snapshots, and checkpoints to a single
// PostgreSQL database via the storage.SQLBackend facade. The event bus uses
// an in-memory implementation (memory.NewMemoryBus) for single-process use.
// For multi-process pub/sub, consumers can wire storage.PostgresBus
// (LISTEN/NOTIFY) via stack.WithBus. Read models persist to the database
// via storage.SQLKVStore.
//
//	b, err := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
//	defer b.Close()
//
// All data is persistent. The returned Bundle owns the *sql.DB; Close releases it.
package postgres
