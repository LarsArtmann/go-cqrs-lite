package sql

import (
	"database/sql"
	"sync/atomic"
)

// Base holds the shared database connection and dialect for all SQL stores.
// Embed this struct in store implementations to avoid duplicating the DB and Dialect fields.
type Base struct {
	DB      *sql.DB
	Dialect Dialect
}

// NewBase creates a Base with the given DB and Dialect, returning ErrNilDB if db is nil.
func NewBase(db *sql.DB, d Dialect) (Base, error) {
	if db == nil {
		return Base{}, ErrNilDB
	}

	return Base{DB: db, Dialect: d}, nil
}

// Close is a no-op for Base (the DB connection lifetime is managed externally).
func (Base) Close() error { return nil }

// ClosableBase extends Base with ownership tracking and a closed state.
// Embed this in stores that need their own Close/checkClosed lifecycle.
type ClosableBase struct {
	Base

	ownDB  bool
	closed atomic.Bool
}

// NewClosableBase creates a ClosableBase with the given DB, Dialect, and ownership flag.
func NewClosableBase(db *sql.DB, d Dialect, ownDB bool) (*ClosableBase, error) {
	base, err := NewBase(db, d)
	if err != nil {
		return nil, err
	}

	return &ClosableBase{Base: base, ownDB: ownDB}, nil
}

// SetOwnership marks the underlying *sql.DB as owned by this base,
// meaning Close will also close the DB connection.
func (b *ClosableBase) SetOwnership(ownDB bool) {
	b.ownDB = ownDB
}

// Close marks the store as closed. If ownDB is true, also closes the underlying *sql.DB.
func (b *ClosableBase) Close() error {
	b.closed.Store(true)

	if b.ownDB {
		return b.DB.Close()
	}

	return nil
}

// CheckClosed returns closedErr if the store has been closed, nil otherwise.
func (b *ClosableBase) CheckClosed(closedErr error) error {
	if b.closed.Load() {
		return closedErr
	}

	return nil
}
