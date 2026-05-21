package storage

import "database/sql"

// SQLEventStoreOption configures an SQLEventStore.
type SQLEventStoreOption func(*SQLEventStore)

// WithOwnership makes the SQLEventStore own the *sql.DB lifecycle.
// When set, Close() will call db.Close(). By default, the DB is borrowed
// and the caller is responsible for closing it.
func WithOwnership() SQLEventStoreOption {
	return func(s *SQLEventStore) {
		s.ownDB = true
	}
}

// NewSQLEventStoreWithOptions creates a new SQL-backed event store with options.
func NewSQLEventStoreWithOptions(db *sql.DB, opts ...SQLEventStoreOption) (*SQLEventStore, error) {
	s, err := NewSQLEventStore(db)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}
