package turso

import "github.com/larsartmann/go-cqrs-lite/core/event"

// ErrTursoMemorySync is returned when trying to sync an in-memory Turso database.
var ErrTursoMemorySync = event.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)
