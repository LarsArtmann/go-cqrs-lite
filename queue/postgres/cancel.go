package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Cancel withdraws a Pending task; the reason rides the cancelled
// fact's detail.
func (s *Store[T]) Cancel(ctx context.Context, id task.ID, reason string) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks SET status = 'cancelled', updated_at = $1, lease_owner = '', lease_expires = NULL
			WHERE id = $2 AND status = 'pending'`, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return statusOrNotFound(ctx, tx, id, "cancelled")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Cancelled, Detail: cancelReasonDetail(reason),
		})
	})
}

// CancelRunning records a cooperative cancel request; the fact IS the
// flag. Idempotent: a second request appends nothing.
func (s *Store[T]) CancelRunning(ctx context.Context, id task.ID, reason string) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		var current string

		if err := tx.QueryRow(ctx,
			`SELECT status FROM tasks WHERE id = $1`, id.String()).Scan(&current); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
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

	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM facts WHERE task_id = $1 AND type = 'task.cancel-requested')`,
		id.String()).Scan(&requested)

	return requested, err
}

// CancelOwned finalizes a cooperative cancel: Running → Cancelled by
// the lease-holding worker, carrying the request's reason.
func (s *Store[T]) CancelOwned(ctx context.Context, id task.ID, owner string) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks SET status = 'cancelled', updated_at = $1, lease_owner = '', lease_expires = NULL
			WHERE id = $2 AND status = 'running' AND lease_owner = $3`,
			now.UnixMilli(), id.String(), owner)
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
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

// MarkOrphaned appends one orphaned fact per stranded Running task
// (lease expired before the cutoff, no orphaned fact yet). Observation
// only; idempotent per task.
func (s *Store[T]) MarkOrphaned(ctx context.Context, cutoff time.Time) (int, error) {
	marked := 0

	err := s.withTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT t.id, t.lease_owner, t.lease_expires
			FROM tasks t
			WHERE t.status = 'running'
			  AND t.lease_expires IS NOT NULL
			  AND t.lease_expires < $1
			  AND NOT EXISTS (
			    SELECT 1 FROM facts f
			    WHERE f.task_id = t.id AND f.type = 'task.orphaned')`,
			cutoff.UnixMilli())
		if err != nil {
			return err
		}
		defer rows.Close()

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

		return s.appendOrphanFacts(ctx, tx, found, &marked)
	})
	if err != nil {
		return 0, err
	}

	return marked, nil
}

// orphan is one stranded Running task observed by MarkOrphaned.
type orphan struct {
	id      string
	owner   string
	expires int64
}

// appendOrphanFacts writes the observed orphans.
func (s *Store[T]) appendOrphanFacts(
	ctx context.Context,
	tx pgx.Tx,
	found []orphan,
	marked *int,
) error {
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

		*marked++
	}

	return nil
}

// RescueDead re-queues a Dead task with a fresh attempt budget; the
// enqueued fact carries the rescue marker.
func (s *Store[T]) RescueDead(ctx context.Context, id task.ID, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = task.DefaultMaxAttempts
	}

	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'pending', attempts = 0, max_attempts = $1, not_before = 0,
			    updated_at = $2, lease_owner = '', lease_expires = NULL, last_error = ''
			WHERE id = $3 AND status = 'dead'`, maxAttempts, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return statusOrNotFound(ctx, tx, id, "pending")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Enqueued,
			Detail: mustJSON(map[string]string{"rescue": "true"}),
		})
	})
}

// DismissDead cancels a Dead task with a recorded reason and by (DLQ
// dismiss).
func (s *Store[T]) DismissDead(ctx context.Context, id task.ID, reason string, dismissedBy string) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'cancelled', updated_at = $1, lease_owner = '', lease_expires = NULL
			WHERE id = $2 AND status = 'dead'`, now.UnixMilli(), id.String())
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return statusOrNotFound(ctx, tx, id, "cancelled")
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Cancelled, Detail: dismissReasonDetail(reason, dismissedBy),
		})
	})
}

// UpdatePendingPriority changes a PENDING task's priority with its
// reprioritized fact in the same transaction; same-value updates append
// nothing.
func (s *Store[T]) UpdatePendingPriority(
	ctx context.Context, id task.ID, newPriority int, source string, reason string,
) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		var status string

		var oldPriority int

		err := tx.QueryRow(ctx, `SELECT status, priority FROM tasks WHERE id = $1`, id.String()).
			Scan(&status, &oldPriority)
		if errors.Is(err, pgx.ErrNoRows) {
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
	ctx context.Context, tx pgx.Tx, id task.ID, oldPriority, newPriority int,
	source, reason string, now time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE tasks SET priority = $1, updated_at = $2
		WHERE id = $3 AND status = 'pending'`, newPriority, now.UnixMilli(), id.String())
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
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
func statusOrNotFound(ctx context.Context, tx pgx.Tx, id task.ID, want string) error {
	var current string

	if err := tx.QueryRow(ctx, `SELECT status FROM tasks WHERE id = $1`, id.String()).
		Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return queue.ErrNotFound
		}

		return err
	}

	return fmt.Errorf("%w: %s -> %s", queue.ErrInvalidTransition, current, want)
}
