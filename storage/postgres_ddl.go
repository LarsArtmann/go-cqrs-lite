package storage

import (
	"context"
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"
)

// postgresDDLAdvisoryKey serializes CQRS schema setup across every session
// touching one PostgreSQL database. Concurrent CREATE TABLE IF NOT EXISTS can
// collide inside PostgreSQL's catalog (pg_type unique-index violations), so
// [PostgresInitSchema] runs under a transaction-scoped advisory lock. The key
// value is shared with metaengine/pgengine and metaengine/claimkit on purpose:
// all CQRS DDL on one database serializes against itself, which makes
// concurrent engine construction safe for t.Parallel test suites and for
// multi-process rolling deploys alike. 0x63717273 spells "cqrs" in ASCII.
const postgresDDLAdvisoryKey = int64(0x63717273)

// execPostgresDDLLocked runs DDL statements inside a single transaction whose
// advisory lock serializes concurrent schema setup database-wide. The lock is
// transaction-scoped, so commit/rollback releases it even on error paths.
func execPostgresDDLLocked(ctx context.Context, db *sql.DB, ddls []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.postgres_ddl_lock", "begin DDL transaction")
	}

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", postgresDDLAdvisoryKey); err != nil {
		_ = tx.Rollback()

		return errorfamily.WrapInfrastructure(err, "storage.postgres_ddl_lock", "acquire advisory lock")
	}

	for _, ddl := range ddls {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			_ = tx.Rollback()

			return errorfamily.WrapInfrastructure(err, "storage.postgres_ddl_lock", "exec DDL: "+ddl)
		}
	}

	if err := tx.Commit(); err != nil {
		return errorfamily.WrapInfrastructure(err, "storage.postgres_ddl_lock", "commit DDL transaction")
	}

	return nil
}
