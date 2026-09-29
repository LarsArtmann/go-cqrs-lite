package pgengine

import (
	"context"
	"database/sql"
	"fmt"
)

// ddlAdvisoryKey serializes CQRS DDL across every session touching one
// PostgreSQL database: concurrent CREATE TABLE IF NOT EXISTS can collide
// inside PostgreSQL's catalog (pg_type unique-index violations). The value is
// shared with storage and metaengine/claimkit on purpose — all CQRS DDL on one
// database serializes against itself, keeping engine construction safe under
// t.Parallel suites and multi-process rolling deploys. 0x63717273 spells
// "cqrs" in ASCII.
const ddlAdvisoryKey = int64(0x63717273)

// execDDLLocked runs DDL statements inside one transaction guarded by a
// database-scoped advisory lock. The lock is transaction-scoped, so
// commit/rollback releases it even on error paths.
func execDDLLocked(ctx context.Context, db *sql.DB, ddls []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine.execDDLLocked: begin: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", ddlAdvisoryKey); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("pgengine.execDDLLocked: advisory lock: %w", err)
	}

	for _, ddl := range ddls {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("pgengine.execDDLLocked: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine.execDDLLocked: commit: %w", err)
	}

	return nil
}
