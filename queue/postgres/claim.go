package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// candidateSQL locks the next claimable row FOR UPDATE SKIP LOCKED —
// competing workers sit on disjoint rows instead of queueing (the
// multi-machine replacement for SQLite's single serialized writer).
// Semantics otherwise match the SQLite engine: status-aware due-ness,
// dependency gating, effective-priority order.
const candidateSQL = `
	SELECT t.id, t.status, t.lease_owner FROM tasks t
	WHERE ((t.status = 'pending' AND t.not_before <= $1)
	    OR (t.status = 'running' AND t.lease_expires IS NOT NULL AND t.lease_expires <= $1))
	  AND NOT EXISTS (
	    SELECT 1 FROM deps d JOIN tasks dt ON dt.id = d.dep_id
	    WHERE d.task_id = t.id AND dt.status != 'completed'
	  )
	ORDER BY t.priority + LEAST(($2 - t.created_at) / 86400000.0 / $3, $4) DESC, t.created_at ASC, t.id ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED`

// claimUpdateSQL stamps the lease; the RowsAffected re-check is the
// fence against a lost race. The minted token is stamped beside the
// lease: it is the holder proof every finalize predicate re-checks
// (ADR-0134).
const claimUpdateSQL = `
	UPDATE tasks
	SET status = 'running', lease_owner = $1, lease_expires = $2, lease_token = $3, updated_at = $4
	WHERE id = $5 AND (
	    (status = 'pending' AND not_before <= $4)
	    OR (status = 'running' AND lease_expires IS NOT NULL AND lease_expires <= $4))`

// ClaimDue atomically claims one due task for owner.
func (s *Store[T]) ClaimDue(
	ctx context.Context,
	owner string,
	lease time.Duration,
) (queue.Claim[T], error) {
	now := time.Now()

	var claimed task.Task[T]

	token := queue.NewClaimToken()

	finalizedCancel := false

	err := s.withTx(ctx, func(tx pgx.Tx) error {
		id, st, prevOwner, err := selectCandidate(ctx, tx, now)
		if err != nil {
			return err
		}

		if st == "running" {
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
	//art-dupl:accept dialect twin of queue/sqlite ClaimDue tail; conformance pins lease semantics
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
	tx pgx.Tx,
	now time.Time,
) (string, string, string, error) {
	row := tx.QueryRow(ctx, candidateSQL,
		now.UnixMilli(), now.UnixMilli(),
		float64(queue.PriorityAgingDaysPerPoint), float64(queue.PriorityAgingMaxBonus))

	var id, status, prevOwner string

	if err := row.Scan(&id, &status, &prevOwner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	ctx context.Context, tx pgx.Tx, id, prevOwner, owner string, now time.Time,
) (bool, error) {
	//art-dupl:accept dialect twin of queue/sqlite finalizeReclaim; conformance pins reclaim semantics
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

	tag, err := tx.Exec(ctx, `
		UPDATE tasks SET status = 'cancelled', updated_at = $1, lease_owner = '', lease_expires = NULL, lease_token = NULL
		WHERE id = $2 AND status = 'running'`, now.UnixMilli(), id)
	if err != nil {
		return false, err
	}

	if tag.RowsAffected() == 0 {
		return false, queue.ErrNoTaskDue // lost the race; another path finalized it
	}

	return true, s.appendFact(ctx, tx, facts.Fact{
		TaskID: id, Type: facts.Cancelled, Owner: owner,
		Detail: cooperativeCancelDetail(reason, "lease-expiry"),
	})
}

// stampLease takes the lease and appends the Claimed fact.
func (s *Store[T]) stampLease(
	ctx context.Context,
	tx pgx.Tx,
	id, owner, token string,
	now time.Time,
	lease time.Duration,
	out *task.Task[T],
) error {
	tag, err := tx.Exec(ctx, claimUpdateSQL,
		owner, now.Add(lease).UnixMilli(), token, now.UnixMilli(), id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
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
