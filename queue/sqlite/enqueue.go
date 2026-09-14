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

	if err := s.insertTask(ctx, tk, n.DedupKey); err != nil {
		return task.Task[T]{}, fmt.Errorf("queue/sqlite: enqueue: %w", err)
	}

	return tk, nil
}

// insertTask writes the task row, its deps rows and the enqueued fact in
// one transaction, re-checking the dedup key inside it (the unique
// partial index is the final arbiter under concurrency).
func (s *Store[T]) insertTask(ctx context.Context, tk task.Task[T], dedupKey string) error {
	depsJSON, err := json.Marshal(tk.Deps)
	if err != nil {
		return fmt.Errorf("marshal deps: %w", err)
	}

	payload, err := s.encodePayload(tk.Payload)
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
				tk.ID = task.ID(existingID)

				return errSuppressed{existing: existingID}
			}

			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}

		if err := s.insertTaskRow(ctx, tx, tk, string(depsJSON), payload, dedupKey); err != nil {
			return err
		}

		for _, d := range tk.Deps {
			if _, err := tx.ExecContext(ctx, `INSERT INTO deps (task_id, dep_id) VALUES (?, ?)`,
				tk.ID.String(), d.String()); err != nil {
				return err
			}
		}

		return s.appendFact(ctx, tx, facts.Fact{
			TaskID: tk.ID.String(), Type: facts.Enqueued, Attempt: 0,
			Detail: mustJSON(map[string]any{"project": tk.Project, "type": tk.Type}),
		})
	})
}

// insertTaskRow runs the tasks-table INSERT.
func (s *Store[T]) insertTaskRow(
	ctx context.Context, tx *sql.Tx, tk task.Task[T], depsJSON string, payload []byte, dedupKey string,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO tasks (id, project, type, payload, deps, priority, attempts, max_attempts,
		                    not_before, status, created_at, updated_at, dedup_key)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, 'pending', ?, ?, ?)`,
		tk.ID.String(), tk.Project, tk.Type, payload, depsJSON, tk.Priority,
		tk.MaxAttempts, ms(tk.NotBefore), tk.CreatedAt.UnixMilli(), tk.UpdatedAt.UnixMilli(), dedupKey)

	return err
}

// errSuppressed marks a dedup-suppressed insert: the tx commits (the
// re-check read must stand) and the caller re-reads the stored task.
type errSuppressed struct{ existing string }

func (errSuppressed) Error() string { return "queue/sqlite: dedup suppressed" }

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

	tk, err := s.Get(ctx, task.ID(id))
	if err != nil {
		return task.Task[T]{}, false, err
	}

	return tk, true, nil
}
