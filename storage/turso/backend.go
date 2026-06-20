package turso

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// Backend is a facade that exposes every SQL-backed CQRS store sharing a
// single Turso/LibSQL database connection. It is the Turso equivalent of
// [storage.SQLBackend] (constructed via [storage.NewSQLiteBackend]) and is
// the recommended one-stop entry point for applications that want the full
// event-sourcing stack: events, commands, queries, snapshots, and
// projection checkpoints.
//
// All store accessors are goroutine-safe and lazily constructed on first
// use. The underlying [*sql.DB] is borrowed from the caller; calling
// [Backend.Close] closes the derived stores but NOT the *sql.DB — the
// caller closes it (or a [SyncDB] owns it).
//
// Quick start:
//
//	db, _ := turso.Open(turso.DbPath("app.db"))
//	turso.ConfigurePool(db)
//	backend, _ := turso.NewBackend(db)
//	defer backend.Close()
//
//	eventStore := backend.EventStore()
//	cmdStore, _ := backend.CommandStore()
//	qStore, _ := backend.QueryStore()
//	snapStore, _ := backend.SnapshotStore()
//	cpStore, _ := backend.CheckpointStore()
type Backend = storage.SQLBackend

// NewBackend creates a [Backend] facade over a Turso/LibSQL database.
//
// It delegates to [storage.NewSQLiteBackend] because Turso's embedded LibSQL
// uses the SQLite dialect. All five stores (event, command, query, snapshot,
// checkpoint) share the provided *sql.DB connection and are created lazily on
// first accessor call.
//
// The *sql.DB is borrowed, not owned — the caller is responsible for closing
// it. [Backend.Close] only closes the derived stores.
func NewBackend(db *sql.DB) (*Backend, error) {
	return storage.NewSQLiteBackend(
		db,
	)
}
