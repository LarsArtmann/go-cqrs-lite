package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Cancel withdraws a Pending task. A non-empty reason is stored in the
// task.cancelled fact detail ("reason" key).
func (s *Store[T]) Cancel(ctx context.Context, id task.ID, reason string) error {
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
		var st string

		if err := tx.QueryRowContext(ctx,
			`SELECT status FROM tasks WHERE id = ?`, id.String()).Scan(&st); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return queue.ErrNotFound
			}

			return err
		}

		if st != "running" {
			return fmt.Errorf("%w: %s -> cancel-requested (only running tasks)", queue.ErrInvalidTransition, st)
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
// written by the lease-holding worker after it stopped the execution.
// The operator's reason (from the cancel-requested fact) is carried onto
// the cancelled fact.
func (s *Store[T]) CancelOwned(ctx context.Context, id task.ID, owner string) error {
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks SET status = 'cancelled', updated_at = ?, lease_owner = '', lease_expires = NULL
			WHERE id = ? AND status = 'running' AND lease_owner = ?`,
			now.UnixMilli(), id.String(), owner)
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
		defer func() { _ = rows.Close() }()

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
func (s *Store[T]) DismissDead(ctx context.Context, id task.ID, reason string, by string) error {
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
			TaskID: id.String(), Type: facts.Cancelled, Detail: dismissReasonDetail(reason, by),
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
	var st string

	if err := tx.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, id.String()).Scan(&st); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return queue.ErrNotFound
		}

		return err
	}

	return fmt.Errorf("%w: %s -> %s", queue.ErrInvalidTransition, st, want)
}

const cancelRequestedSQL = `SELECT EXISTS(
	SELECT 1 FROM facts WHERE task_id = ? AND type = 'task.cancel-requested')`

// cancelRequestedTx is the in-transaction variant of CancelRequested.
func cancelRequestedTx(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var requested bool

	err := tx.QueryRowContext(ctx, cancelRequestedSQL, id).Scan(&requested)

	return requested, err
}

// cancelRequestedReasonTx reads the reason a task's latest cancel
// request carried ("" when none). Best-effort: an unparsable detail
// yields "", never an error — the finalize must not fail on cosmetics.
func cancelRequestedReasonTx(ctx context.Context, tx *sql.Tx, id string) (string, error) {
	var detail string

	err := tx.QueryRowContext(ctx, `
		SELECT detail FROM facts
		WHERE task_id = ? AND type = 'task.cancel-requested'
		ORDER BY seq DESC LIMIT 1`, id).Scan(&detail)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	var d struct {
		Reason string `json:"reason"`
	}

	if json.Unmarshal([]byte(detail), &d) != nil {
		return "", nil
	}

	return d.Reason, nil
}

// cancelReasonDetail builds the detail for a Cancel/CancelRunning fact:
// nil without a reason (no detail noise), {"reason": ...} with one.
func cancelReasonDetail(reason string) []byte {
	if reason == "" {
		return nil
	}

	return mustJSON(map[string]string{"reason": reason})
}

// dismissReasonDetail builds the cancelled detail for a DLQ dismiss: the
// reason plus who ruled. The reason is the point of the dismissal — an
// empty one still records the by.
func dismissReasonDetail(reason string, by string) []byte {
	detail := map[string]string{"dismissed_by": by}
	if reason != "" {
		detail["reason"] = reason
	}

	return mustJSON(detail)
}

// cooperativeCancelDetail builds the cancelled detail for a cooperative
// finalize: the cooperative marker, the finalize context ("after" key,
// when set) and the operator's reason, when one was given.
func cooperativeCancelDetail(reason string, after string) []byte {
	detail := map[string]string{"cooperative": "true"}
	if after != "" {
		detail["after"] = after
	}

	if reason != "" {
		detail["reason"] = reason
	}

	return mustJSON(detail)
}
