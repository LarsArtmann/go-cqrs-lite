package turso

import "github.com/larsartmann/go-cqrs-lite/event/v2"

// ErrMemorySync is returned when trying to sync an in-memory Turso database.
var ErrMemorySync = event.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)
