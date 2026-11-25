package pgengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// lockMapKey serializes MapUpdate callers on one key for the rest of the
// transaction. SELECT ... FOR UPDATE alone does not: when the row does not
// exist yet there is nothing to lock, so two create-on-update transactions
// both read nil and the second upsert overwrites the first result (a lost
// update). The transaction-scoped advisory lock is taken BEFORE the read, so
// the second caller waits and then sees the committed value. It lives in the
// same transaction as the row lock and is released with it.
func lockMapKey(ctx context.Context, q metaengine.SQLExec, namespace, key string) error {
	if _, err := q.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1 || chr(31) || $2))`,
		namespace, key,
	); err != nil {
		return fmt.Errorf("pgengine.MapUpdate: lock key: %w", err)
	}

	return nil
}
