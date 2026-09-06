package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// claimDueMySQL is the MySQL/MariaDB claim path: one
// SELECT ... FOR UPDATE SKIP LOCKED fences every due row, then a plain
// UPDATE stamps the lease inside the same transaction. MySQL-compatible
// servers have no UPDATE..FROM..RETURNING, so the claim needs two
// statements; the row locks from the SELECT keep it atomic — concurrent
// claimers skip the locked rows entirely, so each timer is claimed by
// exactly one poller while its lease is fresh.
//
// SKIP LOCKED requires MySQL 8.0+ or MariaDB 10.6+ (verified live against
// MariaDB 11.4: a transaction holding row locks does not block a concurrent
// SKIP LOCKED claim of the remaining rows). Older servers fail the claim
// query loudly at the first Due — never silently.
func claimDueMySQL[P any](
	ctx context.Context,
	c *ClaimingTimerStore[P],
	tx *sql.Tx,
	now, leaseUntil time.Time,
) ([]scheduling.Timer[P], error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, fire_at, payload FROM timers
WHERE fire_at <= ? AND (lease_until IS NULL OR lease_until <= ?)
ORDER BY fire_at ASC
FOR UPDATE SKIP LOCKED`, c.formatTime(now), c.formatTime(now))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim", "claim due timers")
	}

	timers, joinErr := c.scanClaimed(rows)

	_ = rows.Close()

	if len(timers) > 0 {
		ids := make([]string, len(timers))
		for i, timer := range timers {
			ids[i] = timer.ID.String()
		}

		if err := stampLeaseMySQL(ctx, tx, c.formatTime(leaseUntil), ids); err != nil {
			return nil, err
		}
	}

	return timers, joinErr
}

// stampLeaseMySQL stamps lease_until on the claimed rows inside the claim
// transaction. Only claimed rows are updated: SKIP LOCKED excluded rows
// already locked by others, and our own locks keep everyone else out until
// commit, so exactly the claimed rows are stamped.
func stampLeaseMySQL(
	ctx context.Context,
	tx *sql.Tx,
	leaseUntil string,
	ids []string,
) error {
	args := make([]any, 0, len(ids)+1)
	args = append(args, leaseUntil)
	for _, id := range ids {
		args = append(args, id)
	}

	query := "UPDATE timers SET lease_until = ? WHERE id IN (" +
		strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ")"

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim_stamp", "stamp claimed lease")
	}

	return nil
}

// ensureLeaseColumnMySQL adds the lease_until column when missing. MySQL
// servers have no ADD COLUMN IF NOT EXISTS, so the column is probed via
// information_schema first (works on both MySQL and MariaDB).
func ensureLeaseColumnMySQL(ctx context.Context, db *sql.DB) error {
	var count int

	err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'timers' AND COLUMN_NAME = 'lease_until'`,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("sqlstore: probe lease column: %w", err)
	}

	if count > 0 {
		return nil
	}

	const stmt = `ALTER TABLE timers ADD COLUMN lease_until DATETIME(3) NULL`

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("sqlstore: add lease column: %w", err)
	}

	return nil
}
