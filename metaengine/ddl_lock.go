package metaengine

import (
	"context"
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"
)

// postgresDDLAdvisoryKey serializes CQRS DDL across every session touching
// one PostgreSQL database: concurrent CREATE TABLE IF NOT EXISTS can collide
// inside PostgreSQL's catalog (pg_type unique-index violations). The value is
// shared with storage's PostgresInitSchema on purpose (storage is a separate
// module and keeps its own copy of the constant) — all CQRS DDL on one
// database serializes against itself, keeping engine construction safe under
// t.Parallel suites and multi-process rolling deploys. 0x63717273 spells
// "cqrs" in ASCII.
const postgresDDLAdvisoryKey = int64(0x63717273)

// ExecDDLLocked runs DDL statements inside one transaction guarded by a
// database-scoped advisory lock. The lock is transaction-scoped, so
// commit/rollback releases it even on error paths. The label prefixes every
// error's code and message (e.g. "pgengine.execDDLLocked") so the failing
// caller is identifiable from the classified Infrastructure error alone.
func ExecDDLLocked(ctx context.Context, db *sql.DB, label string, ddls []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, label, "begin DDL transaction")
	}

	if _, err := tx.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock($1)",
		postgresDDLAdvisoryKey,
	); err != nil {
		_ = tx.Rollback()

		return errorfamily.WrapInfrastructure(err, label, "acquire advisory lock")
	}

	for _, ddl := range ddls {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			_ = tx.Rollback()

			return errorfamily.WrapInfrastructure(err, label, "exec DDL: "+ddl)
		}
	}

	if err := tx.Commit(); err != nil {
		return errorfamily.WrapInfrastructure(err, label, "commit DDL transaction")
	}

	return nil
}
