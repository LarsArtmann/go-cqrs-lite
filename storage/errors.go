package storage

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrNilDB is returned when a nil *sql.DB is passed to a storage constructor.
var ErrNilDB = errors.New("storage: nil database connection")

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrNilDB, event.Infrastructure)
}
