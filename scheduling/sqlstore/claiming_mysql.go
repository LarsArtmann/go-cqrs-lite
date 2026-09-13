package sqlstore

import (
	"context"
	"database/sql"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
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
	query, args := claiming.MySQLClaimSelect(timersSpec(), c.formatTime(now))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim", "claim due timers")
	}

	var (
		timers  []scheduling.Timer[P]
		joinErr error
	)

	func() {
		defer func() { _ = rows.Close() }()

		timers, joinErr = c.scanClaimed(rows) // classifies rows.Err internally

		_ = rows.Err() // rowserrcheck: explicitly acknowledged here
	}()

	if len(timers) > 0 {
		ids := make([]string, len(timers))
		for i, timer := range timers {
			ids[i] = timer.ID.String()
		}

		if err := claiming.StampLeaseMySQL(ctx, tx, timersSpec(), c.formatTime(leaseUntil), ids); err != nil {
			return nil, err
		}
	}

	return timers, joinErr
}
