package sql

import _ "embed"

//go:embed migrations/postgres.sql
var postgresSchema string

//go:embed migrations/sqlite.sql
var sqliteSchema string

// PostgresSchemaEmbed returns the full Postgres schema DDL from the embedded
// migrations/postgres.sql file.
func PostgresSchemaEmbed() string { return postgresSchema }

// SQLiteSchemaEmbed returns the full SQLite schema DDL from the embedded
// migrations/sqlite.sql file.
func SQLiteSchemaEmbed() string { return sqliteSchema }
