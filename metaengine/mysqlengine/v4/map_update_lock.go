package mysqlengine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mapLockWaitSeconds bounds how long a MapUpdate waits for another caller
// updating the same key: a lock a dead connection failed to release must
// fail calls loudly instead of hanging them forever.
const mapLockWaitSeconds = 30

// lockMapKey serializes MapUpdate callers on one key and returns the func
// that releases the lock. SELECT ... FOR UPDATE alone does not: when the row
// does not exist yet there is nothing to lock, so two create-on-update
// transactions both read nil and the second upsert overwrites the first
// result (a lost update). The lock is taken BEFORE the read.
//
// MySQL named locks belong to the session, not the transaction, so the
// caller must pass the executor of a transaction (or of an enclosing
// RunInTx): a *sql.Tx pins one connection, which keeps the lock until the
// release func runs on that same connection. Releasing before the commit is
// safe: the next holder then waits on the row lock of the committed insert.
// The name is hashed because servers cap lock names at 64 characters. A
// release that fails leaves the lock on its connection; the wait bound turns
// that into a loud error instead of a hang.
func lockMapKey(ctx context.Context, q metaengine.SQLExec, namespace, key string) (func(), error) {
	sum := sha256.Sum256([]byte(namespace + "\x1f" + key))
	name := fmt.Sprintf("cqrs-map:%x", sum[:16])

	var got sql.NullInt64

	err := q.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", name, mapLockWaitSeconds).Scan(&got)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.MapUpdate: lock key: %w", err)
	}

	if !got.Valid || got.Int64 != 1 {
		return nil, fmt.Errorf(
			"mysqlengine.MapUpdate: lock key: not acquired within %ds", mapLockWaitSeconds)
	}

	// WithoutCancel: a canceled context must not skip the release and leave
	// the lock on a pooled connection.
	releaseCtx := context.WithoutCancel(ctx)

	return func() {
		_, _ = q.ExecContext(releaseCtx, "SELECT RELEASE_LOCK(?)", name)
	}, nil
}
