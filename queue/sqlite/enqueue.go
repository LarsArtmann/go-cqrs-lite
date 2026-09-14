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

// Enqueue persists a new task and records task.enqueued. When
// New.DedupKey is set and a task with that key already exists, the
// stored task is returned unchanged — no duplicate row, no duplicate
// fact (idempotent enqueue).
func (s *Store[T]) Enqueue(ctx context.Context, n task.New[T]) (task.Task[T], error) {
	n = n.Normalize()
	if n.Type == "" {
		return task.Task[T]{}, queue.ErrEmptyType
	}

	if n.DedupKey != "" {
		if existing, found, err := s.getTaskByDedupKey(ctx, n.DedupKey); err != nil {
			return task.Task[T]{}, fmt.Errorf("queue/sqlite: enqueue dedup lookup: %w", err)
		} else if found {
			return existing, nil
		}
	}

	now := time.Now()

	tk := task.Task[T]{
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

	if err := s.insertTask(ctx, &tk, n.DedupKey, &suppressed); err != nil {
		return task.Task[T]{}, fmt.Errorf("queue/sqlite: enqueue: %w", err)
	}

	if suppressed {
		return s.Get(ctx, tk.ID)
	}

	return tk, nil
}

// insertTask writes the task row, its deps rows and the enqueued fact in
// one transaction, re-checking the dedup key inside it (the unique
// partial index is the final arbiter under concurrency). On suppression
// it sets suppressed and leaves t.ID pointing at the stored task.
func (s *Store[T]) insertTask(ctx context.Context, t *task.Task[T], dedupKey string, suppressed *bool) error {
	depsJSON, err := json.Marshal(t.Deps)
	if err != nil {
		return fmt.Errorf("marshal deps: %w", err)
	}

	payload, err := s.encodePayload(t.Payload)
	if err != nil {
		return err
	}

	return s.withTx(ctx, func(tx *sql.Tx) error {
		if dedupKey != "" {
			// Re-check inside the transaction: a concurrent enqueuer may
			// have inserted the same key between our lookup and this write.
			var existingID string

			err := tx.QueryRowContext(ctx, `SELECT id FROM tasks WHERE dedup_key = ?`, dedupKey).Scan(&existingID)
			if err == nil {
				t.ID = task.ID(existingID)
				*suppressed = true

				return nil
			}

			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}

		if err := s.insertTaskRow(ctx, tx, *t, string(depsJSON), payload, dedupKey); err != nil {
			return err
		}

		for _, d := range t.Deps {
			if _, err := tx.ExecContext(ctx, `INSERT INTO deps (task_id, dep_id) VALUES (?, ?)`,
				t.ID.String(), d.String()); err != nil {
				return err
			}
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: t.ID.String(), Type: facts.Enqueued, Attempt: 0,
			Detail: mustJSON(map[string]any{"project": t.Project, "type": t.Type}),
		})
	})
}

// insertTaskRow runs the tasks-table INSERT.
func (s *Store[T]) insertTaskRow(
	ctx context.Context, tx *sql.Tx, t task.Task[T], depsJSON string, payload []byte, dedupKey string,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO tasks (id, project, type, payload, deps, priority, attempts, max_attempts,
		                    not_before, status, created_at, updated_at, dedup_key)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, 'pending', ?, ?, ?)`,
		t.ID.String(), t.Project, t.Type, payload, depsJSON, t.Priority,
		t.MaxAttempts, ms(t.NotBefore), t.CreatedAt.UnixMilli(), t.UpdatedAt.UnixMilli(), dedupKey)

	return err
}

// getTaskByDedupKey returns the stored task for a dedup key, if any.
func (s *Store[T]) getTaskByDedupKey(ctx context.Context, key string) (task.Task[T], bool, error) {
	var id string

	err := s.db.QueryRowContext(ctx, `SELECT id FROM tasks WHERE dedup_key = ?`, key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task[T]{}, false, nil
	}

	if err != nil {
		return task.Task[T]{}, false, err
	}

	t, err := s.Get(ctx, task.ID(id))
	if err != nil {
		return task.Task[T]{}, false, err
	}

	return t, true, nil
}
