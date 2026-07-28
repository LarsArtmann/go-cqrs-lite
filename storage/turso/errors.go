package turso

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMemorySync is returned when trying to sync an in-memory Turso database.
var ErrMemorySync = errorfamily.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)

// wrapInfraOrOK returns nil when err is nil, otherwise wraps err as an
// infrastructure error with the given code and message. Collapses the
// repeated "if err != nil { return WrapInfrastructure(...) }; return nil"
// boilerplate in SyncDB methods — the unique code stays a parameter.
func wrapInfraOrOK(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}
