package storage

import "database/sql"

type SQLEventStoreOption func(*SQLEventStore)

func WithOwnership() SQLEventStoreOption {
	return func(s *SQLEventStore) { s.ownDB = true }
}

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
