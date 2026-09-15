package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Complete marks a Running task Completed.
func (s *Store[T]) Complete(ctx context.Context, id task.ID, owner string, result []byte) error {
	now := time.Now()

	return s.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'completed', completed_at = ?, updated_at = ?,
			    lease_owner = '', lease_expires = NULL, last_error = ''
			WHERE id = ? AND status = 'running' AND lease_owner = ? AND lease_expires > ?`,
			now.UnixMilli(), now.UnixMilli(), id.String(), owner, now.UnixMilli())
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return s.leaseErr(ctx, tx, id)
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Completed, Owner: owner,
			Detail: maybeBytes(result),
		})
	})
}

// Fail records a failed attempt: retry with backoff or dead-letter.
func (s *Store[T]) Fail(
	ctx context.Context,
	id task.ID,
	owner string,
	errText string,
	backoff time.Duration,
	evidence []byte,
) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		now := time.Now() // captured inside the tx: backoff counts from commit, not call

		attempts, maxAttempts, err := readAttempts(ctx, tx, id)
		if err != nil {
			return err
		}

		newAttempts := attempts + 1
		if newAttempts >= maxAttempts {
			return s.deadLetter(ctx, tx, id, owner, newAttempts, errText, evidence, false)
		}

		return s.retryRow(ctx, tx, id, owner, newAttempts, errText, now.Add(backoff), evidence)
	})
}

// readAttempts loads the attempt counters; safe without FOR UPDATE
// under the single serialized writer.
func readAttempts(ctx context.Context, tx *sql.Tx, id task.ID) (int, int, error) {
	var attempts, maxAttempts int

	err := tx.QueryRowContext(ctx,
		`SELECT attempts, max_attempts FROM tasks WHERE id = ?`, id.String()).
		Scan(&attempts, &maxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, queue.ErrNotFound
	}

	if err != nil {
		return 0, 0, err
	}

	return attempts, maxAttempts, nil
}

// retryRow returns the task to Pending with the backoff ladder and
// appends the Failed fact.
func (s *Store[T]) retryRow(
	ctx context.Context, tx *sql.Tx, id task.ID, owner string, newAttempts int, errText string,
	notBefore time.Time, evidence []byte,
) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET status = 'pending', attempts = ?, last_error = ?, not_before = ?,
		    updated_at = ?, lease_owner = '', lease_expires = NULL
		WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		newAttempts, errText, ms(notBefore), time.Now().UnixMilli(), id.String(), owner)
	if err != nil {
		return err
	}

	// A stale owner must not requeue a task it no longer holds (e.g. one
	// parked by a requeue) — gate the fact on the same rows check.
	if n, _ := res.RowsAffected(); n == 0 {
		return s.leaseErr(ctx, tx, id)
	}

	return s.appendFact(ctx, tx, facts.Fact{
		TaskID: id.String(), Type: facts.Failed, Owner: owner,
		Attempt: newAttempts, Error: errText, Detail: maybeBytes(evidence),
	})
}

// deadLetter moves a Running task to Dead and appends the Failed +
// DeadLettered fact pair. permanent also stamps the budget so the
// premature death is visible in the row itself.
func (s *Store[T]) deadLetter(
	ctx context.Context, tx *sql.Tx, id task.ID, owner string, newAttempts int,
	errText string, evidence []byte, permanent bool,
) error {
	class := "exhausted"
	maxStmt := `SET status = 'dead', attempts = ?, last_error = ?, updated_at = ?,
		    lease_owner = '', lease_expires = NULL`
	args := []any{newAttempts, errText, time.Now().UnixMilli()}

	if permanent {
		class = "permanent"
		maxStmt = `SET status = 'dead', attempts = ?, max_attempts = ?, last_error = ?, updated_at = ?,
		    lease_owner = '', lease_expires = NULL`
		args = []any{newAttempts, newAttempts, errText, time.Now().UnixMilli()}
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE tasks `+maxStmt+` WHERE id = ? AND status = 'running' AND lease_owner = ?`,
		append(args, id.String(), owner)...)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return s.leaseErr(ctx, tx, id)
	}

	if err := s.appendFact(ctx, tx, facts.Fact{
		TaskID: id.String(), Type: facts.Failed, Owner: owner,
		Attempt: newAttempts, Error: errText, Detail: maybeBytes(evidence),
	}); err != nil {
		return err
	}

	return s.appendFact(ctx, tx, facts.Fact{
		TaskID: id.String(), Type: facts.DeadLettered, Owner: owner, Attempt: newAttempts,
		Error: errText, Detail: mustJSON(map[string]string{"class": class}),
	})
}

// FailPermanent dead-letters immediately: a permanent error means the
// identical retry would fail identically. The failing attempt is still
// counted.
func (s *Store[T]) FailPermanent(
	ctx context.Context, id task.ID, owner string, errText string, evidence []byte,
) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var attempts int

		err := tx.QueryRowContext(ctx, `SELECT attempts FROM tasks WHERE id = ?`, id.String()).
			Scan(&attempts)
		if errors.Is(err, sql.ErrNoRows) {
			return queue.ErrNotFound
		}

		if err != nil {
			return err
		}

		return s.deadLetter(ctx, tx, id, owner, attempts+1, errText, evidence, true)
	})
}

// Requeue returns a claimed task to Pending without counting an
// attempt: the executor refused to start (preflight), so the task itself
// is fine and the environment is expected to become ready later.
func (s *Store[T]) Requeue(
	ctx context.Context,
	id task.ID,
	owner string,
	errText string,
	delay time.Duration,
) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		now := time.Now()

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'pending', not_before = ?, last_error = ?, updated_at = ?,
			    lease_owner = '', lease_expires = NULL
			WHERE id = ? AND status = 'running' AND lease_owner = ?`,
			ms(now.Add(delay)), errText, now.UnixMilli(), id.String(), owner)
		if err != nil {
			return err
		}

		if n, _ := res.RowsAffected(); n == 0 {
			return s.leaseErr(ctx, tx, id)
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Requeued, Owner: owner, Error: errText,
			Detail: mustJSON(facts.RequeueEvidence{Reason: errText, RetryIn: delay.Milliseconds()}),
		})
	})
}

// Heartbeat extends the lease of a Running task held by owner.
func (s *Store[T]) Heartbeat(
	ctx context.Context,
	id task.ID,
	owner string,
	extend time.Duration,
) error {
	now := time.Now()

	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET lease_expires = ?, updated_at = ?
		WHERE id = ? AND status = 'running' AND lease_owner = ? AND lease_expires > ?`,
		ms(now.Add(extend)), now.UnixMilli(), id.String(), owner, now.UnixMilli())
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return queue.ErrLeaseNotHeld
	}

	return nil
}

// leaseErr classifies a zero-rows finalize: not-found when the task is
// gone, ErrLeaseNotHeld otherwise.
func (s *Store[T]) leaseErr(ctx context.Context, q taskQuerier, id task.ID) error {
	var st string

	err := q.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, id.String()).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return queue.ErrNotFound
	}

	if err != nil {
		return err
	}

	return queue.ErrLeaseNotHeld
}
