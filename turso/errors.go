package turso

import "github.com/larsartmann/go-cqrs-lite/event"

// ErrMemorySync is returned when trying to sync an in-memory Turso database.
var ErrMemorySync = event.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)

// ErrTursoMemorySync is a backward-compatible alias for ErrMemorySync.
var ErrTursoMemorySync = ErrMemorySync
