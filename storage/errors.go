package storage

import "github.com/cockroachdb/errors"

// ErrNilDB is returned when a nil *sql.DB is passed to a storage constructor.
var ErrNilDB = errors.New("storage: nil database connection")
