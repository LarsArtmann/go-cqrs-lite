package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// claimDeadlockRetries bounds the in-engine retry of a claim
// transaction that InnoDB killed with a deadlock (1213) or lock-wait
// timeout (1205). Concurrent two-statement claims touch index-gap locks
// in nondeterministic order, so occasional deadlocks are NORMAL on
// MySQL — retrying the whole transaction is the documented InnoDB
// remedy, and the claim is idempotent from the caller's perspective
// (either it takes the lease or reports ErrNoTaskDue).
const claimDeadlockRetries = 3

// candidateSQL locks the next claimable row FOR UPDATE SKIP LOCKED —
// competing workers sit on disjoint rows instead of queueing (MySQL 8+/
// MariaDB 10.6+). Semantics match the other engines: status-aware
// due-ness, dependency gating, effective-priority order (aging first,
// cap included), oldest-first tie-break.
const candidateSQL = `
	SELECT t.id, t.status, t.lease_owner FROM tasks t
	WHERE ((t.status = 'pending' AND t.not_before <= ?)
	    OR (t.status = 'running' AND t.lease_expires IS NOT NULL AND t.lease_expires <= ?))
	  AND NOT EXISTS (
	    SELECT 1 FROM deps d JOIN tasks dt ON dt.id = d.dep_id
	    WHERE d.task_id = t.id AND dt.status != 'completed'
	  )
	ORDER BY t.priority + LEAST((? - t.created_at) / 86400000.0 / ?, ?) DESC, t.created_at ASC, t.id ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED`

// claimUpdateSQL stamps the lease; the RowsAffected re-check is the
// fence against a lost race. The minted token is stamped beside the
// lease: it is the holder proof every finalize predicate re-checks
// (ADR-0134).
const claimUpdateSQL = `
	UPDATE tasks
	SET status = 'running', lease_owner = ?, lease_expires = ?, lease_token = ?, updated_at = ?
	WHERE id = ? AND (
	    (status = 'pending' AND not_before <= ?)
	    OR (status = 'running' AND lease_expires IS NOT NULL AND lease_expires <= ?))`

// ClaimDue atomically claims one due task for owner. InnoDB may
// deadlock the claim transaction under concurrency (normal for
// two-statement claims); the engine retries it internally before
// surfacing the error.
func (s *Store[T]) ClaimDue(
	ctx context.Context,
	owner string,
	lease time.Duration,
) (queue.Claim[T], error) {
	var (
		claim queue.Claim[T]
		err   error
	)

	for range claimDeadlockRetries + 1 {
		claim, err = s.claimOnce(ctx, owner, lease)
		if err == nil || !isDeadlock(err) {
			break
		}
	}

	return claim, err
}

// isDeadlock reports whether err is InnoDB's deadlock (1213) or
// lock-wait timeout (1205) signal.
func isDeadlock(err error) bool {
	var myErr *mysqldriver.MySQLError

	if errors.As(err, &myErr) {
		return myErr.Number == 1213 || myErr.Number == 1205
	}

	return false
}

// claimOnce runs one claim attempt (the transaction the retry loop
// re-runs on deadlock).
func (s *Store[T]) claimOnce(
	ctx context.Context,
	owner string,
	lease time.Duration,
) (queue.Claim[T], error) {
	now := time.Now()

	var claimed task.Task[T]

	token := queue.NewClaimToken()

	finalizedCancel := false

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		id, st, prevOwner, err := selectCandidate(ctx, tx, now)
		if err != nil {
			return err
		}

		if st == "running" {
			// Reclaiming an expired lease. A pending cooperative cancel is
			// finalized here instead: the task is never re-executed after
			// its cancel was requested. Either way the journal records the
			// release, so it explains the owner change.
			done, err := s.finalizeReclaim(ctx, tx, id, prevOwner, owner, now)
			if err != nil {
				return err
			}

			if done {
				finalizedCancel = true

				return nil
			}
		}

		return s.stampLease(ctx, tx, id, owner, token, now, lease, &claimed)
	})
	//art-dupl:accept dialect twin of queue/sqlite+postgres ClaimDue tail; conformance pins lease semantics
	if err != nil {
		return queue.Claim[T]{}, err
	}

	if finalizedCancel {
		return queue.Claim[T]{}, queue.ErrNoTaskDue
	}

	// Truncate to the stored millisecond so the surfaced deadline is
	// exactly the persisted one.
	return queue.Claim[T]{
		Task:       claimed,
		LeaseUntil: time.UnixMilli(now.Add(lease).UnixMilli()),
		Token:      token,
	}, nil
}

// selectCandidate runs the locking candidate query.
func selectCandidate(
	ctx context.Context,
	tx *sql.Tx,
	now time.Time,
) (string, string, string, error) {
	row := tx.QueryRowContext(ctx, candidateSQL,
		now.UnixMilli(), now.UnixMilli(),
		float64(now.UnixMilli()),
		float64(queue.PriorityAgingDaysPerPoint), float64(queue.PriorityAgingMaxBonus))

	var id, status, prevOwner string

	if err := row.Scan(&id, &status, &prevOwner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", queue.ErrNoTaskDue
		}

		return "", "", "", err
	}

	return id, status, prevOwner, nil
}

// finalizeReclaim handles an expired-lease candidate: a pending
// cooperative cancel is finalized (the task never runs again); either
// way the Released fact explains the owner change.
func (s *Store[T]) finalizeReclaim(
	ctx context.Context, tx *sql.Tx, id, prevOwner, owner string, now time.Time,
) (bool, error) {
	//art-dupl:accept dialect twin of queue/sqlite+postgres finalizeReclaim; conformance pins reclaim semantics
	requested, err := cancelRequestedTx(ctx, tx, id)
	if err != nil {
		return false, err
	}

	if err := s.appendFact(
		ctx,
		tx,
		facts.Fact{TaskID: id, Type: facts.Released, Owner: prevOwner},
	); err != nil {
		return false, err
	}

	if !requested {
		return false, nil
	}

	reason, err := cancelRequestedReasonTx(ctx, tx, id)
	if err != nil {
		return false, err
	}

	if err := s.cancelRunningRow(ctx, tx, id, now); err != nil {
		return false, err
	}

	return true, s.appendFact(ctx, tx, facts.Fact{
		TaskID: id, Type: facts.Cancelled, Owner: owner,
		Detail: cooperativeCancelDetail(reason, "lease-expiry"),
	})
}

// cancelRunningRow flips one running row to cancelled (the reclaim
// finalize path).
func (s *Store[T]) cancelRunningRow(
	ctx context.Context,
	tx *sql.Tx,
	id string,
	now time.Time,
) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks SET status = 'cancelled', updated_at = ?, lease_owner = '', lease_expires = NULL, lease_token = NULL
		WHERE id = ? AND status = 'running'`, now.UnixMilli(), id)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return queue.ErrNoTaskDue // lost the race; another path finalized it
	}

	return nil
}

// stampLease takes the lease and appends the Claimed fact; on success it
// loads the claimed row into out.
func (s *Store[T]) stampLease(
	ctx context.Context,
	tx *sql.Tx,
	id, owner, token string,
	now time.Time,
	lease time.Duration,
	out *task.Task[T],
) error {
	res, err := tx.ExecContext(
		ctx,
		claimUpdateSQL,
		owner,
		now.Add(lease).UnixMilli(),
		token,
		now.UnixMilli(),
		id,
		now.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return queue.ErrNoTaskDue // lost the race; caller retries
	}

	if err := s.appendFact(
		ctx,
		tx,
		facts.Fact{TaskID: id, Type: facts.Claimed, Owner: owner},
	); err != nil {
		return err
	}

	claimed, err := s.loadTaskTx(ctx, tx, id)
	if err != nil {
		return err
	}

	*out = claimed

	return nil
}
