package turso

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMemorySync is returned when trying to sync an in-memory Turso database.
var ErrMemorySync = errorfamily.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)
