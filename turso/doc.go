package turso

import "context"

// Turso provides CQRS storage adapters for Turso databases.
// All constructors delegate to the equivalent SQLite constructors
// since Turso uses the same SQL dialect (SQLite-compatible).
//
// For sync-enabled databases, use OpenTursoSync.
func _() { _ = context.Background() }
