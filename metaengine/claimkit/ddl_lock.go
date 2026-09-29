package claimkit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
)

// ddlAdvisoryKey serializes CQRS DDL across every session touching one
// PostgreSQL database: concurrent CREATE TABLE IF NOT EXISTS can collide
// inside PostgreSQL's catalog (pg_type unique-index violations). The value is
// shared with storage and metaengine/pgengine on purpose — all CQRS DDL on one
// database serializes against itself, keeping engine construction safe under
// t.Parallel suites and multi-process rolling deploys. 0x63717273 spells
// "cqrs" in ASCII.
const ddlAdvisoryKey = int64(0x63717273)

// ensureClaimsTablesLocked runs the Postgres claims DDL (tables + indexes)
// inside one transaction guarded by a database-scoped advisory lock. The lock
// is transaction-scoped, so commit/rollback releases it even on error paths.
func ensureClaimsTablesLocked(ctx context.Context, db *sql.DB, d claiming.Dialect) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("claimkit.ensureClaimsTablesLocked: begin: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", ddlAdvisoryKey); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("claimkit.ensureClaimsTablesLocked: advisory lock: %w", err)
	}

	for _, ddl := range claimsDDL(d) {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("claimkit.ensureClaimsTablesLocked: ensure table: %w", err)
		}
	}

	for _, ddl := range []string{
		`CREATE INDEX IF NOT EXISTS idx_meta_due_claims_due ON meta_due_claims(collection, due_at)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_claim_facts_key ON meta_claim_facts(collection, key, seq)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_dedup_expiry ON meta_dedup(collection, expires_at)`,
	} {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("claimkit.ensureClaimsTablesLocked: ensure index: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("claimkit.ensureClaimsTablesLocked: commit: %w", err)
	}

	return nil
}
