package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Cancel withdraws a Pending task. A non-empty reason is stored in the
// task.cancelled fact detail ("reason" key).
func (s *Store[T]) Cancel(ctx context.Context, id task.ID, reason string) error {
	//art-dupl:accept dialect twin — queue engines are dep-isolated mirrors; conformance pins cancel semantics
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks SET status = 'cancelled', updated_at = ?, lease_owner = '', lease_expires = NULL
			WHERE id = ? AND status = 'pending'`, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return statusOrNotFound(ctx, tx, id, "cancelled")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Cancelled, Detail: cancelReasonDetail(reason),
		})
	})
}

// CancelRunning records a cooperative cancel request for a Running
// task. The task.cancel-requested fact IS the flag — no task-row column
// mirrors it (facts-first). Idempotent: a second request appends
// nothing.
func (s *Store[T]) CancelRunning(ctx context.Context, id task.ID, reason string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var current string

		if err := tx.QueryRowContext(ctx,
			`SELECT status FROM tasks WHERE id = ?`, id.String()).Scan(&current); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return queue.ErrNotFound
			}

			return err
		}

		if current != "running" {
			return fmt.Errorf(
				"%w: %s -> cancel-requested (only running tasks)",
				queue.ErrInvalidTransition,
				current,
			)
		}

		requested, err := cancelRequestedTx(ctx, tx, id.String())
		if err != nil {
			return err
		}

		if requested {
			return nil // already requested; the flag is the fact
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.CancelRequested, Detail: cancelReasonDetail(reason),
		})
	})
}

// CancelRequested reports whether a cooperative cancel request is
// pending.
func (s *Store[T]) CancelRequested(ctx context.Context, id task.ID) (bool, error) {
	var requested bool

	err := s.db.QueryRowContext(ctx, cancelRequestedSQL, id.String()).Scan(&requested)

	return requested, err
}

// CancelOwned finalizes a cooperative cancel: Running → Cancelled,
// written by the claim-holding worker (claim token required) after it
// stopped the execution. The operator's reason (from the
// cancel-requested fact) is carried onto the cancelled fact.
func (s *Store[T]) CancelOwned(ctx context.Context, id task.ID, token string) error {
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		owner, err := leaseHolder(ctx, tx, id)
		if err != nil {
			return err
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks SET status = 'cancelled', updated_at = ?, lease_owner = '', lease_expires = NULL, lease_token = NULL
			WHERE id = ? AND status = 'running' AND lease_token = ?`,
			now.UnixMilli(), id.String(), token)
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return queue.ErrLeaseNotHeld
		}

		reason, err := cancelRequestedReasonTx(ctx, tx, id.String())
		if err != nil {
			return err
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Cancelled, Owner: owner,
			Detail: cooperativeCancelDetail(reason, ""),
		})
	})
}

// MarkOrphaned appends one task.orphaned fact per stranded Running task
// (lease expired before the cutoff, no orphaned fact yet). Observation
// only: the task stays Running until a reclaim. Idempotent per task.
func (s *Store[T]) MarkOrphaned(ctx context.Context, cutoff time.Time) (int, error) {
	marked := 0

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		//nolint:sqlclosecheck // rows closed via deferred DeferClose below
		rows, err := tx.QueryContext(ctx, `
			SELECT t.id, COALESCE(t.lease_owner, ''), t.lease_expires
			FROM tasks t
			WHERE t.status = 'running'
			  AND t.lease_expires IS NOT NULL
			  AND t.lease_expires < ?
			  AND NOT EXISTS (
			    SELECT 1 FROM facts f
			    WHERE f.task_id = t.id AND f.type = 'task.orphaned')`,
			cutoff.UnixMilli())
		if err != nil {
			return err
		}
		defer metaengine.DeferClose(rows)

		type orphan struct {
			id      string
			owner   string
			expires int64
		}

		var found []orphan

		for rows.Next() {
			var o orphan

			if err := rows.Scan(&o.id, &o.owner, &o.expires); err != nil {
				return err
			}

			found = append(found, o)
		}

		if err := rows.Err(); err != nil {
			return err
		}

		for _, o := range found {
			detail := mustJSON(map[string]any{
				"owner":          o.owner,
				"leaseExpiredAt": time.UnixMilli(o.expires).UTC().Format(time.RFC3339),
			})

			if err := s.appendFact(ctx, tx, facts.Fact{
				TaskID: o.id, Type: facts.Orphaned, Owner: o.owner, Detail: detail,
			}); err != nil {
				return err
			}

			marked++
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return marked, nil
}

// RescueDead re-queues a Dead task with a fresh attempt budget (DLQ
// rescue); the enqueued fact carries the rescue marker.
func (s *Store[T]) RescueDead(ctx context.Context, id task.ID, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = task.DefaultMaxAttempts
	}

	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'pending', attempts = 0, max_attempts = ?, not_before = 0,
			    updated_at = ?, lease_owner = '', lease_expires = NULL, last_error = ''
			WHERE id = ? AND status = 'dead'`, maxAttempts, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return statusOrNotFound(ctx, tx, id, "pending")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Enqueued,
			Detail: mustJSON(map[string]string{"rescue": "true"}),
		})
	})
}

// DismissDead cancels a Dead task with a recorded reason (DLQ dismiss):
// the cancelled fact's detail carries the reason and who dismissed it.
func (s *Store[T]) DismissDead(
	ctx context.Context,
	id task.ID,
	reason string,
	dismissedBy string,
) error {
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'cancelled', updated_at = ?, lease_owner = '', lease_expires = NULL
			WHERE id = ? AND status = 'dead'`, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return statusOrNotFound(ctx, tx, id, "cancelled")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(),
			Type:   facts.Cancelled,
			Detail: dismissReasonDetail(reason, dismissedBy),
		})
	})
}

// UpdatePendingPriority changes a PENDING task's priority; the
// reprioritized fact (old/new, source, reason) lands in the SAME
// transaction, and a same-value update appends nothing.
func (s *Store[T]) UpdatePendingPriority(
	ctx context.Context, id task.ID, newPriority int, source string, reason string,
) error {
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		var status string

		var oldPriority int

		err := tx.QueryRowContext(ctx, `SELECT status, priority FROM tasks WHERE id = ?`, id.String()).
			Scan(&status, &oldPriority)
		if errors.Is(err, sql.ErrNoRows) {
			return queue.ErrNotFound
		}

		if err != nil {
			return err
		}

		if status != "pending" {
			return fmt.Errorf("%w: %s priority change", queue.ErrInvalidTransition, status)
		}

		if oldPriority == newPriority {
			return nil // idempotent: same value, no fact
		}

		return s.updatePriorityRow(ctx, tx, id, oldPriority, newPriority, source, reason, now)
	})
}

// updatePriorityRow writes the guarded UPDATE plus its fact.
func (s *Store[T]) updatePriorityRow(
	ctx context.Context, tx *sql.Tx, id task.ID, oldPriority, newPriority int,
	source, reason string, now time.Time,
) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks SET priority = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`, newPriority, now.UnixMilli(), id.String())
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: pending priority change", queue.ErrInvalidTransition)
	}

	return s.appendFact(ctx, tx, facts.Fact{
		TaskID: id.String(), Type: facts.Reprioritized,
		Detail: mustJSON(facts.ReprioritizeEvidence{
			OldPriority: oldPriority, NewPriority: newPriority, Source: source, Reason: reason,
		}),
	})
}

// statusOrNotFound maps a zero-rows guarded update to the right error.
func statusOrNotFound(ctx context.Context, tx *sql.Tx, id task.ID, want string) error {
	var current string

	if err := tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, id.String()).
		Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queue.ErrNotFound
		}

		return err
	}

	return fmt.Errorf("%w: %s -> %s", queue.ErrInvalidTransition, current, want)
}
