package pgengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// lockStream takes a transaction-scoped advisory lock for one stream, held
// until the surrounding transaction ends. The version check in
// StreamAppendExpected is only sound while nobody else can append to the
// stream between the check and the insert.
func lockStream(ctx context.Context, q metaengine.SQLExec, col, sid string) error {
	if _, err := q.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1 || chr(31) || $2))`,
		col, sid,
	); err != nil {
		return fmt.Errorf("pgengine.StreamAppendExpected lock: %w", err)
	}

	return nil
}
