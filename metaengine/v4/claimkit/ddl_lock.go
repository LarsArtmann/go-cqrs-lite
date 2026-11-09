package claimkit

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// claimsIndexDDL carries the Postgres index DDL every claims schema needs;
// kept next to ensureClaimsTablesLocked so the table+index set reads as one.
var claimsIndexDDL = []string{
	`CREATE INDEX IF NOT EXISTS idx_meta_due_claims_due ON meta_due_claims(collection, due_at)`,
	`CREATE INDEX IF NOT EXISTS idx_meta_claim_facts_key ON meta_claim_facts(collection, key, seq)`,
	`CREATE INDEX IF NOT EXISTS idx_meta_dedup_expiry ON meta_dedup(collection, expires_at)`,
}

// ensureClaimsTablesLocked runs the Postgres claims DDL (tables + indexes)
// inside one transaction guarded by a database-scoped advisory lock shared by
// all CQRS DDL (see metaengine.ExecDDLLocked).
func ensureClaimsTablesLocked(ctx context.Context, db *sql.DB, d claiming.Dialect) error {
	ddls := append(claimsDDL(d), claimsIndexDDL...)

	return metaengine.ExecDDLLocked(ctx, db, "claimkit.ensureClaimsTablesLocked", ddls)
}
