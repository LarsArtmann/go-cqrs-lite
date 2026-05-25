package storage

import (
	"database/sql"
	"fmt"
)

type sqlBase struct {
	db      *sql.DB
	dialect Dialect
}

func newSQLBase(db *sql.DB, d Dialect) (sqlBase, error) {
	if db == nil {
		return sqlBase{}, fmt.Errorf("%w", ErrNilDB)
	}

	return sqlBase{db: db, dialect: d}, nil
}

func (sqlBase) Close() error { return nil }
