package sql

import (
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

// NewOwnedDBHandle creates an OwnedDBHandle with the given DB, Dialect, and ownership flag.
func NewOwnedDBHandle(db *sql.DB, d Dialect, ownDB bool) (*OwnedDBHandle, error) {
	handle, err := NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &OwnedDBHandle{DBHandle: handle, ownDB: ownDB}, nil
}

// SetOwnership marks the underlying *sql.DB as owned by this handle,
// meaning Close will also close the DB connection.
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
