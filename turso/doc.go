package turso

// Turso provides CQRS storage adapters for Turso databases.
// All constructors delegate to the equivalent SQLite constructors
// since Turso uses the same SQL dialect (SQLite-compatible).
//
// For sync-enabled databases, use OpenTursoSync.
//
// For auto-smart index management, see the turso/indexing sub-package.
// Use InitSchemaWithIndexes to create tables plus CQRS-optimized indexes
// in a single call, or ApplyCQRSIndexes for existing databases.
