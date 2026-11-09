package pgengine

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// execDDLLocked runs DDL statements inside one transaction guarded by the
// database-scoped advisory lock shared by all CQRS DDL — the pgengine-local
// front for metaengine.ExecDDLLocked (the label keeps the failing caller
// identifiable in the classified Infrastructure error).
func execDDLLocked(ctx context.Context, db *sql.DB, ddls []string) error {
	return metaengine.ExecDDLLocked(ctx, db, "pgengine.execDDLLocked", ddls)
}
