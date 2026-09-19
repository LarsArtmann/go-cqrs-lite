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

// Complete marks a Running task Completed. The token predicate is the
// fence: only the current claim holder can finish (ADR-0134).
func (s *Store[T]) Complete(ctx context.Context, id task.ID, token string, result []byte) error {
	now := time.Now()

	return s.withTx(ctx, func(tx pgx.Tx) error {
		owner, err := leaseHolder(ctx, tx, id)
		if err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'completed', completed_at = $1, updated_at = $2,
			    lease_owner = '', lease_expires = NULL, lease_token = NULL, last_error = ''
			WHERE id = $3 AND status = 'running' AND lease_token = $4 AND lease_expires > $2`,
			now.UnixMilli(), now.UnixMilli(), id.String(), token)
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return queue.ErrLeaseNotHeld
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Completed, Owner: owner,
			Detail: maybeBytes(result),
		})
	})
}

// leaseHolder reads the current lease owner (row-locked for the tx's
// duration) for fact attribution — the finalize methods carry the token,
// not the owner, so the journal's attribution comes from the row itself.
func leaseHolder(ctx context.Context, q rower, id task.ID) (string, error) {
	var owner string

	err := q.QueryRow(ctx,
		`SELECT lease_owner FROM tasks WHERE id = $1 FOR UPDATE`, id.String()).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", queue.ErrNotFound
	}

	if err != nil {
		return "", err
	}

	return owner, nil
}

// Fail records a failed attempt: retry with backoff or dead-letter.
func (s *Store[T]) Fail(
	ctx context.Context,
	id task.ID,
	token string,
	errText string,
	backoff time.Duration,
	evidence []byte,
) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		now := time.Now()

		attempts, maxAttempts, owner, err := readAttemptHolder(ctx, tx, id)
		if err != nil {
			return err
		}

		newAttempts := attempts + 1
		if newAttempts >= maxAttempts {
			return s.deadLetter(ctx, tx, id, owner, token, newAttempts, errText, evidence, false)
		}

		return s.retryRow(ctx, tx, id, owner, token, newAttempts, errText, now.Add(backoff), evidence)
	})
}

// readAttemptHolder loads the attempt counters plus the lease owner,
// locking the row for the tx's duration (multi-writer discipline; the
// SQLite twin relies on its single serialized writer instead).
func readAttemptHolder(ctx context.Context, tx pgx.Tx, id task.ID) (int, int, string, error) {
	var attempts, maxAttempts int

	var owner string

	err := tx.QueryRow(ctx,
		`SELECT attempts, max_attempts, lease_owner FROM tasks WHERE id = $1 FOR UPDATE`, id.String()).
		Scan(&attempts, &maxAttempts, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, "", queue.ErrNotFound
	}

	if err != nil {
		return 0, 0, "", err
	}

	return attempts, maxAttempts, owner, nil
}

// retryRow returns the task to Pending with the backoff ladder and
// appends the Failed fact. The token predicate fences the write to the
// current holder.
func (s *Store[T]) retryRow(
	ctx context.Context, tx pgx.Tx, id task.ID, owner, token string, newAttempts int,
	errText string, notBefore time.Time, evidence []byte,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE tasks
		SET status = 'pending', attempts = $1, last_error = $2, not_before = $3,
		    updated_at = $4, lease_owner = '', lease_expires = NULL, lease_token = NULL
		WHERE id = $5 AND status = 'running' AND lease_token = $6`,
		newAttempts, errText, ms(notBefore), time.Now().UnixMilli(), id.String(), token)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return queue.ErrLeaseNotHeld
	}

	return s.appendFact(ctx, tx, facts.Fact{
		TaskID: id.String(), Type: facts.Failed, Owner: owner,
		Attempt: newAttempts, Error: errText, Detail: maybeBytes(evidence),
	})
}

// deadLetter moves a Running task to Dead and appends the Failed +
// DeadLettered fact pair. permanent also stamps the budget so the
// premature death is visible in the row itself. The token predicate
// fences the write to the current holder.
func (s *Store[T]) deadLetter(
	ctx context.Context, tx pgx.Tx, id task.ID, owner, token string, newAttempts int,
	errText string, evidence []byte, permanent bool,
) error {
	class := "exhausted"
	setStmt := `SET status = 'dead', attempts = $1, last_error = $2, updated_at = $3,
		    lease_owner = '', lease_expires = NULL, lease_token = NULL WHERE id = $4 AND status = 'running' AND lease_token = $5`
	args := []any{newAttempts, errText, time.Now().UnixMilli(), id.String(), token}

	if permanent {
		class = "permanent"
		setStmt = `SET status = 'dead', attempts = $1, max_attempts = $2, last_error = $3, updated_at = $4,
		    lease_owner = '', lease_expires = NULL, lease_token = NULL WHERE id = $5 AND status = 'running' AND lease_token = $6`
		args = []any{newAttempts, newAttempts, errText, time.Now().UnixMilli(), id.String(), token}
	}

	tag, err := tx.Exec(ctx, `UPDATE tasks `+setStmt, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return queue.ErrLeaseNotHeld
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
	ctx context.Context, id task.ID, token string, errText string, evidence []byte,
) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		var attempts int

		var owner string

		err := tx.QueryRow(ctx,
			`SELECT attempts, lease_owner FROM tasks WHERE id = $1 FOR UPDATE`, id.String()).
			Scan(&attempts, &owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return queue.ErrNotFound
		}

		if err != nil {
			return err
		}

		return s.deadLetter(ctx, tx, id, owner, token, attempts+1, errText, evidence, true)
	})
}

// Requeue returns a claimed task to Pending without counting an
// attempt (preflight refusal).
func (s *Store[T]) Requeue(
	ctx context.Context,
	id task.ID,
	token string,
	errText string,
	delay time.Duration,
) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		now := time.Now()

		owner, err := leaseHolder(ctx, tx, id)
		if err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status = 'pending', not_before = $1, last_error = $2, updated_at = $3,
			    lease_owner = '', lease_expires = NULL, lease_token = NULL
			WHERE id = $4 AND status = 'running' AND lease_token = $5`,
			ms(now.Add(delay)), errText, now.UnixMilli(), id.String(), token)
		if err != nil {
			return err
		}

		if tag.RowsAffected() == 0 {
			return queue.ErrLeaseNotHeld
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: id.String(), Type: facts.Requeued, Owner: owner, Error: errText,
			Detail: mustJSON(facts.RequeueEvidence{Reason: errText, RetryIn: delay.Milliseconds()}),
		})
	})
}

// Heartbeat extends the lease of a Running task while the claim token is
// held; no fact is appended (renewal is scheduling, not state).
func (s *Store[T]) Heartbeat(
	ctx context.Context,
	id task.ID,
	token string,
	extend time.Duration,
) error {
	now := time.Now()

	tag, err := s.pool.Exec(ctx, `
		UPDATE tasks SET lease_expires = $1, updated_at = $2
		WHERE id = $3 AND status = 'running' AND lease_token = $4 AND lease_expires > $2`,
		ms(now.Add(extend)), now.UnixMilli(), id.String(), token)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return queue.ErrLeaseNotHeld
	}

	return nil
}
