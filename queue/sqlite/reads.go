package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// taskColumns is the canonical tasks-table read list.
const taskColumns = `id, project, type, payload, deps, priority, attempts, max_attempts,
	             not_before, status, lease_owner, lease_expires, last_error,
	             created_at, updated_at, completed_at`

// Get returns the current task record.
func (s *Store[T]) Get(ctx context.Context, id task.ID) (task.Task[T], error) {
	return s.loadTaskTx(ctx, s.db, id.String())
}

// loadTaskTx loads one task through any query surface.
func (s *Store[T]) loadTaskTx(ctx context.Context, q taskQuerier, id string) (task.Task[T], error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)

	t, err := s.scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return task.Task[T]{}, queue.ErrNotFound
		}

		return task.Task[T]{}, err
	}

	return t, nil
}

// scanner is the shared Scan surface of Row and Rows.
type scanner interface{ Scan(dest ...any) error }

// scanTaskRow decodes one tasks row into a Task through the codec.
func (s *Store[T]) scanTaskRow(r scanner) (task.Task[T], error) {
	var (
		t            task.Task[T]
		id           string
		payload      []byte
		deps         string
		notBeforeMS  int64
		status       string
		leaseExpires sql.NullInt64
		completedAt  sql.NullInt64
		createdAtMS  int64
		updatedAtMS  int64
	)

	if err := r.Scan(&id, &t.Project, &t.Type, &payload, &deps, &t.Priority, &t.Attempts,
		&t.MaxAttempts, &notBeforeMS, &status, &t.LeaseOwner, &leaseExpires, &t.LastError,
		&createdAtMS, &updatedAtMS, &completedAt); err != nil {
		return task.Task[T]{}, err
	}

	t.ID = task.ID(id)
	t.Status = task.Status(status)
	t.NotBefore = time.UnixMilli(notBeforeMS)
	t.CreatedAt = time.UnixMilli(createdAtMS)
	t.UpdatedAt = time.UnixMilli(updatedAtMS)

	if leaseExpires.Valid {
		le := time.UnixMilli(leaseExpires.Int64)
		t.LeaseExpires = &le
	}

	if completedAt.Valid {
		ca := time.UnixMilli(completedAt.Int64)
		t.CompletedAt = &ca
	}

	if deps != "" && deps != "[]" {
		if err := json.Unmarshal([]byte(deps), &t.Deps); err != nil {
			return task.Task[T]{}, fmt.Errorf("queue/sqlite: unmarshal deps for %s: %w", id, err)
		}
	}

	decoded, err := s.codec.Decode(payload)
	if err != nil {
		return task.Task[T]{}, fmt.Errorf("queue/sqlite: decode payload for %s: %w", id, err)
	}

	t.Payload = decoded

	return t, nil
}

// listWhere builds the shared WHERE clause for List and CountTasks so
// the two can never disagree about what a filter matches.
func listWhere(f queue.Filter) (string, []any) {
	where := []string{"1=1"}
	args := []any{}

	if f.Project != nil {
		where = append(where, "project = ?")
		args = append(args, *f.Project)
	}

	if f.Status != nil {
		where = append(where, "status = ?")
		args = append(args, string(*f.Status))
	}

	if f.Type != nil {
		where = append(where, "type = ?")
		args = append(args, *f.Type)
	}

	if f.Since != nil {
		where = append(where, "created_at >= ?")
		args = append(args, f.Since.UnixMilli())
	}

	if f.Parked != nil && *f.Parked {
		where = append(where, "status = 'pending' AND not_before > ?")
		args = append(args, time.Now().UnixMilli())
	}

	if f.PriorityMin != nil {
		where = append(where, "priority >= ?")
		args = append(args, *f.PriorityMin)
	}

	if f.PriorityMax != nil {
		where = append(where, "priority <= ?")
		args = append(args, *f.PriorityMax)
	}

	if f.Query != "" {
		like := "%" + escapeLike(strings.ToLower(f.Query)) + "%"

		where = append(where, `(id LIKE ? ESCAPE '\' OR type LIKE ? ESCAPE '\' OR
			project LIKE ? ESCAPE '\' OR payload LIKE ? ESCAPE '\' OR
			lease_owner LIKE ? ESCAPE '\' OR last_error LIKE ? ESCAPE '\')`)
		args = append(args, like, like, like, like, like, like)
	}

	return strings.Join(where, " AND "), args
}

// List returns tasks matching the filter, ordered by stored priority
// descending then creation age ascending.
func (s *Store[T]) List(ctx context.Context, f queue.Filter) ([]task.Task[T], error) {
	where, args := listWhere(f)

	q := `SELECT ` + taskColumns + ` FROM tasks WHERE ` + where + `
	      ORDER BY priority DESC, created_at ASC`

	if f.Limit > 0 || f.Offset > 0 {
		if f.Limit > 0 {
			q += " LIMIT ?"
			args = append(args, f.Limit)
		} else {
			q += " LIMIT -1"
		}

		if f.Offset > 0 {
			q += " OFFSET ?"
			args = append(args, f.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []task.Task[T]

	for rows.Next() {
		t, err := s.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, t)
	}

	return out, rows.Err()
}

// CountTasks counts the tasks matching the filter (COUNT(*) pushdown).
func (s *Store[T]) CountTasks(ctx context.Context, f queue.Filter) (int, error) {
	where, args := listWhere(f)

	var n int

	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE `+where, args...).Scan(&n)

	return n, err
}

// StatusCounts counts tasks per status in one GROUP BY.
func (s *Store[T]) StatusCounts(ctx context.Context) (map[task.Status]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[task.Status]int)

	for rows.Next() {
		var (
			st task.Status
			n  int
		)

		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}

		out[st] = n
	}

	return out, rows.Err()
}

// escapeLike escapes LIKE wildcards so a user query containing %, _ or \
// matches literally. Pair with ESCAPE '\' in the SQL.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)

	return s
}
