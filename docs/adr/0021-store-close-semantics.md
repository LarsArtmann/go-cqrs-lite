# ADR-0021: Store Close() Semantics — Shared DB Pattern

**Date:** 2026-06-16  
**Status:** Accepted

## Context

When multiple stores share a single `*pebble.DB` (or `*sql.DB`), calling
`Close()` on one store should NOT close the underlying database — other stores
and the application may still be using it.

This creates an asymmetry that needs documentation:

- `EventStore.Close()` marks the store as closed but does NOT close `*pebble.DB`.
- `SnapshotStore.Close()` — same: marks closed, does NOT close `*pebble.DB`.
- `CheckpointStore.Close()` — same.
- `SQLBackend.Close()` closes all child stores but does NOT close `*sql.DB`.

The caller who opened the database connection owns its lifecycle.

## Decision

All stores follow the **borrowed handle** pattern:

1. The caller opens the database (`pebble.Open`, `sql.Open`).
2. The caller passes the `*pebble.DB` / `*sql.DB` to store constructors.
3. Store `Close()` only marks the store as closed — subsequent operations
   return `ErrClosed`.
4. The caller closes the database when all stores are done.

### PebbleBackend Pattern

When using `PebbleBackend.Open()`, the backend owns the `*pebble.DB`:

```go
backend, _ := pebble.Open(dir, &pebble.Options{})
defer backend.Close() // closes the *pebble.DB AND all stores
```

### SQLBackend Pattern

`SQLBackend` borrows the `*sql.DB` — the caller manages its lifecycle:

```go
db, _ := sql.Open("sqlite3", ":memory:")
backend, _ := storage.NewSQLiteBackend(db)
defer backend.Close()   // closes all stores, NOT the db
defer db.Close()        // caller closes the DB
```

## Consequences

- No double-close panics from shared connections.
- Clear ownership: whoever opens the DB closes it.
- `OwnedDBHandle` (SQL) and future pebble backend track `ownDB` flag.
- Documented in store doc.go comments and AGENTS.md.
