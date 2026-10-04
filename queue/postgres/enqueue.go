package postgres

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Enqueue persists a new task and records task.enqueued; dedup keys make
// it idempotent (the partial unique index is the arbiter).
func (s *Store[T]) Enqueue(ctx context.Context, n task.New[T]) (task.Task[T], error) {
	//art-dupl:accept dialect twin of queue/sqlite Enqueue flow; conformance pins idempotency
	n = n.Normalize()
	if n.Type == "" {
		return task.Task[T]{}, queue.ErrEmptyType
	}

	if n.DedupKey != "" {
		if existing, found, err := s.getTaskByDedupKey(ctx, n.DedupKey); err != nil {
			return task.Task[T]{}, fmt.Errorf("queue/postgres: enqueue dedup lookup: %w", err)
		} else if found {
			return existing, nil
		}
	}

	now := time.Now()

	t := task.Task[T]{
		ID:          task.NewID(),
		Project:     n.Project,
		Type:        n.Type,
		Payload:     n.Payload,
		Deps:        n.Deps,
		Priority:    n.Priority,
		MaxAttempts: n.MaxAttempts,
		NotBefore:   n.NotBefore,
		Status:      task.Pending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	suppressed := false

	if err := s.insertTask(ctx, &t, n.DedupKey, &suppressed); err != nil {
		return task.Task[T]{}, fmt.Errorf("queue/postgres: enqueue: %w", err)
	}

	if suppressed {
		return s.Get(ctx, t.ID)
	}

	return t, nil
}

// insertTask writes the task row, its deps rows and the enqueued fact in
// one transaction, re-checking the dedup key inside it.
func (s *Store[T]) insertTask(
	ctx context.Context,
	t *task.Task[T],
	dedupKey string,
	suppressed *bool,
) error {
	depsJSON, err := json.Marshal(t.Deps)
	if err != nil {
		return fmt.Errorf("marshal deps: %w", err)
	}

	payload, err := s.encodePayload(t.Payload)
	if err != nil {
		return err
	}

	return s.withTx(ctx, func(tx pgx.Tx) error {
		if dedupKey != "" {
			var existingID string

			err := tx.QueryRow(ctx, `SELECT id FROM tasks WHERE dedup_key = $1`, dedupKey).
				Scan(&existingID)
			if err == nil {
				t.ID = task.ID(existingID)
				*suppressed = true

				return nil
			}

			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}

		if err := validateDeps(ctx, tx, string(depsJSON)); err != nil {
			return err
		}

		return s.insertRows(ctx, tx, *t, string(depsJSON), payload, dedupKey)
	})
}

// insertRows runs the tasks/deps INSERTs plus the enqueued fact.
func (s *Store[T]) insertRows(
	ctx context.Context,
	tx pgx.Tx,
	t task.Task[T],
	depsJSON string,
	payload string,
	dedupKey string,
) error {
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO tasks (id, project, type, payload, deps, priority, attempts, max_attempts,
		                    not_before, status, created_at, updated_at, dedup_key)
		 VALUES ($1, $2, $3, $4, $5, $6, 0, $7, $8, 'pending', $9, $10, $11)`,
		t.ID.String(),
		t.Project,
		t.Type,
		payload,
		depsJSON,
		t.Priority,
		t.MaxAttempts,
		ms(t.NotBefore),
		t.CreatedAt.UnixMilli(),
		t.UpdatedAt.UnixMilli(),
		dedupKey,
	); err != nil {
		return err
	}

	for _, d := range t.Deps {
		if _, err := tx.Exec(ctx, `INSERT INTO deps (task_id, dep_id) VALUES ($1, $2)`,
			t.ID.String(), d.String()); err != nil {
			return err
		}
	}

	return s.appendFact(ctx, tx, facts.Fact{
		TaskID: t.ID.String(), Type: facts.Enqueued, Attempt: 0,
		Detail: mustJSON(map[string]any{"project": t.Project, "type": t.Type}),
	})
}

// getTaskByDedupKey returns the stored task for a dedup key, if any.
func (s *Store[T]) getTaskByDedupKey(ctx context.Context, key string) (task.Task[T], bool, error) {
	var id string

	err := s.pool.QueryRow(ctx, `SELECT id FROM tasks WHERE dedup_key = $1`, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task[T]{}, false, nil
	}

	if err != nil {
		return task.Task[T]{}, false, err
	}

	t, err := s.Get(ctx, task.ID(id))

	return t, err == nil, err
}

// validateDeps enforces enqueue-time dep existence: every dep ID in the
// JSON array must reference an existing task row. The anti-join over
// jsonb_array_elements_text yields exactly the missing IDs in one query;
// an empty array is trivially valid (and skipped). This is also the
// cycle guard — see queue.ErrDanglingDep.
func validateDeps(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, depsJSON string,
) error {
	if depsJSON == "" || depsJSON == "[]" {
		return nil
	}

	rows, err := q.Query(ctx, `
		SELECT d.value::text FROM jsonb_array_elements_text($1::jsonb) AS d(value)
		LEFT JOIN tasks t ON t.id = d.value
		WHERE t.id IS NULL`, depsJSON)
	if err != nil {
		return fmt.Errorf("validate deps: %w", err)
	}
	defer rows.Close()

	//art-dupl:accept dialect twin of queue/sqlite validateDeps missing-scan; dep-isolated modules
	var missing []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("validate deps: scan: %w", err)
		}

		missing = append(missing, id)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("validate deps: %w", err)
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", queue.ErrDanglingDep, strings.Join(missing, ", "))
	}

	return nil
}
