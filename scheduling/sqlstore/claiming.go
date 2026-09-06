package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// DefaultClaimLease is how long a claim lasts when the operator does not
// name a duration. A claimed timer becomes claimable again only after the
// lease expires, so the lease bounds how long a crashed dispatcher delays a
// timer — and how long concurrent dispatchers are guaranteed not to
// double-fire it.
const DefaultClaimLease = time.Minute

// ErrClaimingUnsupported is returned when a claiming store is requested for
// a dialect that cannot honor the claim contract. MySQL/MariaDB 10.6+
// (FOR UPDATE SKIP LOCKED — verified live on MariaDB 11.4) IS supported;
// only unknown dialects are rejected.
var ErrClaimingUnsupported = errors.New(
	"sqlstore: claiming store requires Postgres, SQLite, or MySQL/MariaDB 10.6+ (FOR UPDATE SKIP LOCKED)",
)

// ClaimingTimerStore wraps a [SQLTimerStore] with atomic claiming so MULTIPLE
// Scheduler instances can share one timers table without double-firing:
//
//   - Due claims due timers by stamping lease_until (SKIP LOCKED on Postgres,
//     writer serialization on SQLite) and returns only the rows IT claimed.
//   - MarkFired deletes after dispatch (unchanged).
//   - A dispatcher that crashes after claiming keeps its timers leased until
//     the lease expires; then another poller may claim them again. This is
//     at-least-once with a no-double-fire guarantee INSIDE the lease window —
//     dispatch handlers should still be idempotent.
//
// The base store's plain Due is bypassed; Schedule, Cancel, MarkFired and
// Close pass through. The timers table gains a lease_until column (added
// idempotently at construction, including for tables created by older
// versions).
type ClaimingTimerStore[P any] struct {
	*SQLTimerStore[P]

	lease   time.Duration
	metrics ClaimMetrics
}

// NewClaimingPostgresStore creates a Postgres-backed claiming timer store.
// lease is how long a claim fences other pollers (DefaultClaimLease when 0).
// The caller retains ownership of db.
func NewClaimingPostgresStore[P any](
	ctx context.Context,
	db *sql.DB,
	lease time.Duration,
	opts ...ClaimOption[P],
) (*ClaimingTimerStore[P], error) {
	return newClaimingStore[P](ctx, db, DialectPostgres, lease, opts...)
}

// NewClaimingSQLiteStore creates a SQLite-backed claiming timer store. SQLite
// has no SKIP LOCKED, but its single-writer model serializes claim
// transactions, which is equivalent for the no-double-fire guarantee.
func NewClaimingSQLiteStore[P any](
	ctx context.Context,
	db *sql.DB,
	lease time.Duration,
	opts ...ClaimOption[P],
) (*ClaimingTimerStore[P], error) {
	return newClaimingStore[P](ctx, db, DialectSQLite, lease, opts...)
}

// NewClaimingMySQLStore creates a MySQL/MariaDB-backed claiming timer store.
// Claiming uses FOR UPDATE SKIP LOCKED, which requires MySQL 8.0+ or
// MariaDB 10.6+; older servers fail the claim query loudly at the first
// Due. lease is how long a claim fences other pollers (DefaultClaimLease
// when 0). The caller retains ownership of db.
func NewClaimingMySQLStore[P any](
	ctx context.Context,
	db *sql.DB,
	lease time.Duration,
	opts ...ClaimOption[P],
) (*ClaimingTimerStore[P], error) {
	return newClaimingStore[P](ctx, db, DialectMySQL, lease, opts...)
}

func newClaimingStore[P any](
	ctx context.Context,
	db *sql.DB,
	d Dialect,
	lease time.Duration,
	opts ...ClaimOption[P],
) (*ClaimingTimerStore[P], error) {
	if d != DialectPostgres && d != DialectSQLite && d != DialectMySQL {
		return nil, ErrClaimingUnsupported
	}

	base, err := newStore[P](ctx, db, d)
	if err != nil {
		return nil, err
	}

	if err := ensureLeaseColumn(ctx, db, d); err != nil {
		return nil, err
	}

	if lease <= 0 {
		lease = DefaultClaimLease
	}

	c := &ClaimingTimerStore[P]{
		SQLTimerStore: base,
		lease:         lease,
		metrics:       ClaimMetrics{Claimed: nil, Renewed: nil, RenewRejected: nil},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Due atomically claims every timer whose fire_at has passed and whose lease
// (if any) has expired, and returns only the claimed timers. Two concurrent
// callers never receive the same timer while its lease is fresh.
func (c *ClaimingTimerStore[P]) Due(
	ctx context.Context,
	now time.Time,
) ([]scheduling.Timer[P], error) {
	leaseUntil := now.Add(c.lease)

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim_tx", "begin claim transaction")
	}
	defer func() { _ = tx.Rollback() }()

	var (
		timers  []scheduling.Timer[P]
		joinErr error
	)

	if c.dialect == DialectMySQL {
		timers, joinErr = claimDueMySQL(ctx, c, tx, now, leaseUntil)
	} else {
		query, args := c.claimStmt(now, leaseUntil)

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "scheduling.sqlstore.claim", "claim due timers")
		}

		timers, joinErr = c.scanClaimed(rows)

		_ = rows.Close()
	}

	if err := tx.Commit(); err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim_commit", "commit claim transaction")
	}

	if c.metrics.Claimed != nil {
		c.metrics.Claimed(len(timers))
	}

	return timers, joinErr
}

// MarkFired removes a timer after dispatch (leases do not protect a fired
// timer from deletion).
func (c *ClaimingTimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	return c.SQLTimerStore.MarkFired(ctx, id)
}

func (c *ClaimingTimerStore[P]) claimStmt(now, leaseUntil time.Time) (string, []any) {
	if c.dialect == DialectPostgres {
		// SKIP LOCKED: concurrent claimers skip each other's locked rows, so
		// each row is claimed by exactly one poller. The lease row predicate
		// re-opens timers whose claim expired (crashed dispatcher).
		return `WITH due AS (
	SELECT id FROM timers
	WHERE fire_at <= $1 AND (lease_until IS NULL OR lease_until <= $1)
	ORDER BY fire_at ASC
	FOR UPDATE SKIP LOCKED
)
UPDATE timers t SET lease_until = $2 FROM due WHERE t.id = due.id
RETURNING t.id, t.fire_at, t.payload`,
			[]any{c.formatTime(now), c.formatTime(leaseUntil)}
	}

	// SQLite: single writer, so a plain UPDATE..RETURNING inside the
	// transaction is already atomic across claimers.
	return `UPDATE timers SET lease_until = ?1
WHERE fire_at <= ?2 AND (lease_until IS NULL OR lease_until <= ?2)
RETURNING id, fire_at, payload`,
		[]any{c.formatTime(leaseUntil), c.formatTime(now)}
}

func (c *ClaimingTimerStore[P]) scanClaimed(
	rows *sql.Rows,
) ([]scheduling.Timer[P], error) {
	var timers []scheduling.Timer[P]

	var corrupt []error

	for rows.Next() {
		var rawID string

		var payload []byte

		dest := c.scanTimeDest()

		if err := rows.Scan(&rawID, dest, &payload); err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "scheduling.sqlstore.claim_scan", "scan claimed timer row")
		}

		fireAt, err := c.parseTime(dest)
		if err != nil {
			return nil, err
		}

		timer, err := decodeDueTimer[P](rawID, payload, fireAt)
		if err != nil {
			corrupt = append(corrupt, err)

			continue
		}

		timers = append(timers, timer)
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.claim_iter", "iterate claimed timers")
	}

	return timers, errors.Join(corrupt...)
}

// ensureLeaseColumn adds lease_until to tables created before claiming
// existed. Idempotent per dialect (Postgres IF NOT EXISTS; SQLite probed via
// pragma_table_info).
func ensureLeaseColumn(ctx context.Context, db *sql.DB, d Dialect) error {
	var stmt string

	switch d {
	case DialectPostgres:
		stmt = `ALTER TABLE timers ADD COLUMN IF NOT EXISTS lease_until TIMESTAMP WITH TIME ZONE`
	case DialectSQLite:
		var count int

		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info('timers') WHERE name = 'lease_until'`,
		).Scan(&count); err != nil {
			return fmt.Errorf("sqlstore: probe lease column: %w", err)
		}

		if count > 0 {
			return nil
		}

		stmt = `ALTER TABLE timers ADD COLUMN lease_until TEXT`
	case DialectMySQL:
		return ensureLeaseColumnMySQL(ctx, db)
	default:
		return ErrClaimingUnsupported
	}

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("sqlstore: add lease column: %w", err)
	}

	return nil
}

// ErrLeaseNotHeld is returned by RenewLease when the caller no longer owns
// the timer's claim: either the timer was fired/canceled (row gone) or the
// lease expired and another poller may have re-claimed it. Classified as
// Orchestration — a distributed-coordination race, not a caller bug.
var ErrLeaseNotHeld = errorfamily.NewOrchestration(
	"scheduling.sqlstore.lease_not_held",
	"no live claim for this timer (fired, canceled, or lease expired)",
)

// RenewLease extends the live claim on a timer whose dispatch handler is
// still running, for handlers that can outlive the lease. The new deadline
// is now+extension, granted only when the lease has NOT yet expired — an
// expired claim cannot be resurrected (another poller may already be
// dispatching), so renewal after expiry returns [ErrLeaseNotHeld] and the
// handler must stop work: at-least-once semantics mean its result may be
// duplicated by whoever re-claimed the timer.
//
// Ownership note: claims are not attributed to a poller (the TimerStore
// interface has no claim tokens), so renewal extends WHICHEVER live claim
// exists. This is the safe direction — a stale owner can only EXTEND the
// fence, which delays re-claiming; it can never cause a double fire.
// Per-poller ownership proof would need claim tokens and is future work.
//
// extension <= 0 means DefaultClaimLease.
func (c *ClaimingTimerStore[P]) RenewLease(
	ctx context.Context,
	id scheduling.TimerID,
	extension time.Duration,
) error {
	if extension <= 0 {
		extension = DefaultClaimLease
	}

	now := time.Now().UTC()
	newUntil := now.Add(extension)

	var query string

	var args []any

	if c.dialect == DialectPostgres {
		query = `UPDATE timers SET lease_until = $1 WHERE id = $2 AND lease_until > $3`
		args = []any{c.formatTime(newUntil), id.String(), c.formatTime(now)}
	} else if c.dialect == DialectMySQL {
		// MySQL has no ordinal ?N placeholders — plain ? only.
		query = `UPDATE timers SET lease_until = ? WHERE id = ? AND lease_until > ?`
		args = []any{c.formatTime(newUntil), id.String(), c.formatTime(now)}
	} else {
		query = `UPDATE timers SET lease_until = ?1 WHERE id = ?2 AND lease_until > ?3`
		args = []any{c.formatTime(newUntil), id.String(), c.formatTime(now)}
	}

	res, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.renew_lease", "renew lease for timer "+id.String())
	}

	n, err := res.RowsAffected()
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "scheduling.sqlstore.renew_lease_rows", "count renewed rows")
	}

	if n == 0 {
		if c.metrics.RenewRejected != nil {
			c.metrics.RenewRejected()
		}

		return fmt.Errorf("sqlstore: renew %s: %w", id.String(), ErrLeaseNotHeld)
	}

	if c.metrics.Renewed != nil {
		c.metrics.Renewed()
	}

	return nil
}
