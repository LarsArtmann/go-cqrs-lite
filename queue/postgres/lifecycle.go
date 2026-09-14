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

// Complete marks a Running task Completed.
func (s *Store[T]) Complete(ctx context.Context, id task.ID, owner string, result []byte) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'completed', completed_at = $1, updated_at = $2,
			    lease_owner = '', lease_expires = NULL, last_error = ''
			WHERE id = $3 AND status = 'running' AND lease_owner = $4 AND lease_expires > $2`,
			now.UnixMilli(), now.UnixMilli(), id.String(), owner)
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
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
	return s.withTx(ctx, func(tx pgx.Tx) error {
		now := time.Now()

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

// readAttempts loads the attempt counters, locking the row for the
// tx's duration (multi-writer discipline; the SQLite twin relies on its
// single serialized writer instead).
func readAttempts(ctx context.Context, tx pgx.Tx, id task.ID) (int, int, error) {
	var attempts, maxAttempts int

	err := tx.QueryRow(ctx,
		`SELECT attempts, max_attempts FROM tasks WHERE id = $1 FOR UPDATE`, id.String()).
		Scan(&attempts, &maxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
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
	ctx context.Context, tx pgx.Tx, id task.ID, owner string, newAttempts int,
	errText string, notBefore time.Time, evidence []byte,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE tasks
		SET status = 'pending', attempts = $1, last_error = $2, not_before = $3,
		    updated_at = $4, lease_owner = '', lease_expires = NULL
		WHERE id = $5 AND status = 'running' AND lease_owner = $6`,
		newAttempts, errText, ms(notBefore), time.Now().UnixMilli(), id.String(), owner)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
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
	ctx context.Context, tx pgx.Tx, id task.ID, owner string, newAttempts int,
	errText string, evidence []byte, permanent bool,
) error {
	class := "exhausted"
	setStmt := `SET status = 'dead', attempts = $1, last_error = $2, updated_at = $3,
		    lease_owner = '', lease_expires = NULL WHERE id = $4 AND status = 'running' AND lease_owner = $5`
	args := []any{newAttempts, errText, time.Now().UnixMilli(), id.String(), owner}

	if permanent {
		class = "permanent"
		setStmt = `SET status = 'dead', attempts = $1, max_attempts = $2, last_error = $3, updated_at = $4,
		    lease_owner = '', lease_expires = NULL WHERE id = $5 AND status = 'running' AND lease_owner = $6`
		args = []any{newAttempts, newAttempts, errText, time.Now().UnixMilli(), id.String(), owner}
	}

	tag, err := tx.Exec(ctx, `UPDATE tasks `+setStmt, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
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

// FailPermanent dead-letters immediately: the error class makes
// retrying pointless. The failing attempt is still counted.
func (s *Store[T]) FailPermanent(
	ctx context.Context, id task.ID, owner string, errText string, evidence []byte,
) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		var attempts int

		err := tx.QueryRow(ctx,
			`SELECT attempts FROM tasks WHERE id = $1 FOR UPDATE`, id.String()).Scan(&attempts)
		if errors.Is(err, pgx.ErrNoRows) {
			return queue.ErrNotFound
		}

		if err != nil {
			return err
		}

		return s.deadLetter(ctx, tx, id, owner, attempts+1, errText, evidence, true)
	})
}

// Requeue returns a claimed task to Pending without counting an
// attempt (preflight refusal).
func (s *Store[T]) Requeue(ctx context.Context, id task.ID, owner string, errText string, delay time.Duration) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		now := time.Now()

		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'pending', not_before = $1, last_error = $2, updated_at = $3,
			    lease_owner = '', lease_expires = NULL
			WHERE id = $4 AND status = 'running' AND lease_owner = $5`,
			ms(now.Add(delay)), errText, now.UnixMilli(), id.String(), owner)
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return s.leaseErr(ctx, tx, id)
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Requeued, Owner: owner, Error: errText,
			Detail: mustJSON(facts.RequeueEvidence{Reason: errText, RetryIn: delay.Milliseconds()}),
		})
	})
}

// Heartbeat extends the lease of a Running task held by owner.
func (s *Store[T]) Heartbeat(ctx context.Context, id task.ID, owner string, extend time.Duration) error {
	now := time.Now()

	tag, err := s.pool.Exec(ctx, `
		UPDATE tasks SET lease_expires = $1, updated_at = $2
		WHERE id = $3 AND status = 'running' AND lease_owner = $4 AND lease_expires > $2`,
		ms(now.Add(extend)), now.UnixMilli(), id.String(), owner)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return queue.ErrLeaseNotHeld
	}

	return nil
}

// leaseErr classifies a zero-rows finalize: not-found when the task is
// gone, ErrLeaseNotHeld otherwise.
func (s *Store[T]) leaseErr(ctx context.Context, q rower, id task.ID) error {
	var st string

	err := q.QueryRow(ctx, `SELECT status FROM tasks WHERE id = $1`, id.String()).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return queue.ErrNotFound
	}

	if err != nil {
		return err
	}

	return queue.ErrLeaseNotHeld
}
