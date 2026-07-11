package sql

import (
	"context"
	"database/sql"
	"sync/atomic"
)

// DBHandle holds the shared database connection and dialect for all SQL stores.
// Embed this struct in store implementations to avoid duplicating the DB and Dialect fields.
type DBHandle struct {
	DB      *sql.DB
	Dialect Dialect
}

// NewDBHandle creates a DBHandle with the given DB and Dialect, returning ErrNilDB if db is nil.
func NewDBHandle(db *sql.DB, d Dialect) (DBHandle, error) {
	if db == nil {
		return DBHandle{}, ErrNilDB
	}

	return DBHandle{DB: db, Dialect: d}, nil
}

// Close is a no-op for DBHandle (the DB connection lifetime is managed externally).
func (DBHandle) Close() error { return nil }

// OwnedDBHandle extends DBHandle with ownership tracking and a closed state.
// Embed this in stores that need their own Close/checkClosed lifecycle.
type OwnedDBHandle struct {
	DBHandle

	ownDB  bool
	closed atomic.Bool
}

// NewBorrowedDBHandle creates an OwnedDBHandle that tracks closed state
// but does NOT close the underlying *sql.DB on Close. The caller retains
// ownership of the DB connection. This is the default for all SQL stores
// that share a backend-managed connection.
func NewBorrowedDBHandle(db *sql.DB, d Dialect) (*OwnedDBHandle, error) {
	return newOwnedDBHandle(db, d, false)
}

// NewOwningDBHandle creates an OwnedDBHandle whose Close will also close
// the underlying *sql.DB. Use this when the handle is the sole owner of
// the connection (e.g. standalone stores that create their own *sql.DB).
func NewOwningDBHandle(db *sql.DB, d Dialect) (*OwnedDBHandle, error) {
	return newOwnedDBHandle(db, d, true)
}

func newOwnedDBHandle(db *sql.DB, d Dialect, ownDB bool) (*OwnedDBHandle, error) {
	handle, err := NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &OwnedDBHandle{DBHandle: handle, ownDB: ownDB}, nil
}

// Deprecated: Use [NewBorrowedDBHandle] or [NewOwningDBHandle] instead.
// The ownDB bool flag makes Close behavior ambiguous at the call site.
func NewOwnedDBHandle(db *sql.DB, d Dialect, ownDB bool) (*OwnedDBHandle, error) {
	return newOwnedDBHandle(db, d, ownDB)
}

// Deprecated: Use [NewOwningDBHandle] at construction time instead.
// Mutating ownership after construction can cause double-close or connection
// leaks if the lifecycle assumption changes mid-flight.
func (b *OwnedDBHandle) SetOwnership(ownDB bool) {
	b.ownDB = ownDB
}

// Close marks the store as closed. If ownDB is true, also closes the underlying *sql.DB.
func (b *OwnedDBHandle) Close() error {
	b.closed.Store(true)

	if b.ownDB {
		return b.DB.Close()
	}

	return nil
}

// CheckClosed returns closedErr if the store has been closed, nil otherwise.
func (b *OwnedDBHandle) CheckClosed(closedErr error) error {
	if b.closed.Load() {
		return closedErr
	}

	return nil
}

// HealthCheck verifies the database connection is alive via PingContext.
// Implements the stack.HealthChecker interface for Kubernetes liveness/readiness probes.
// All stores that embed OwnedDBHandle inherit this method automatically.
func (b *OwnedDBHandle) HealthCheck(ctx context.Context) error {
	if err := b.CheckClosed(ErrClosed); err != nil {
		return err
	}

	return b.DB.PingContext(ctx)
}
