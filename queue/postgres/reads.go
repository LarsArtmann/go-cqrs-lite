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
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// taskColumns is the canonical tasks-table read list.
const taskColumns = `id, project, type, payload, deps, priority, attempts, max_attempts,
                     not_before, status, lease_owner, lease_expires, last_error,
                     created_at, updated_at, completed_at`

// Get returns the current task record.
func (s *Store[T]) Get(ctx context.Context, id task.ID) (task.Task[T], error) {
	t, err := s.scanTask(
		s.pool.QueryRow(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id.String()),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task[T]{}, queue.ErrNotFound
	}

	return t, err
}

// loadTaskTx loads one task through a tx.
func (s *Store[T]) loadTaskTx(ctx context.Context, tx pgx.Tx, id string) (task.Task[T], error) {
	return s.scanTask(tx.QueryRow(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id))
}

// rowScanner is the shared Scan surface of pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanTask decodes one tasks row through the codec (deps rehydrate from
// the deps JSON column, like the SQLite engine).
func (s *Store[T]) scanTask(row rowScanner) (task.Task[T], error) {
	var (
		t            task.Task[T]
		id           string
		depsJSON     string
		payload      string
		leaseExpires *int64
		completedAt  *int64
		notBefore    int64
		createdAt    int64
		updatedAt    int64
	)

	if err := row.Scan(&id, &t.Project, &t.Type, &payload, &depsJSON, &t.Priority,
		&t.Attempts, &t.MaxAttempts, &notBefore, &t.Status, &t.LeaseOwner,
		&leaseExpires, &t.LastError, &createdAt, &updatedAt, &completedAt); err != nil {
		return task.Task[T]{}, err
	}

	t.ID = task.ID(id)
	t.NotBefore = time.UnixMilli(notBefore)
	t.CreatedAt = time.UnixMilli(createdAt)
	t.UpdatedAt = time.UnixMilli(updatedAt)

	if err := json.Unmarshal(
		[]byte(depsJSON),
		&t.Deps,
	); err != nil && depsJSON != "" &&
		depsJSON != "[]" {
		return task.Task[T]{}, fmt.Errorf("queue/postgres: decode deps: %w", err)
	}

	if leaseExpires != nil {
		le := time.UnixMilli(*leaseExpires)
		t.LeaseExpires = &le
	}

	if completedAt != nil {
		ca := time.UnixMilli(*completedAt)
		t.CompletedAt = &ca
	}

	decoded, err := s.codec.Decode([]byte(payload))
	if err != nil {
		return task.Task[T]{}, fmt.Errorf("queue/postgres: decode payload for %s: %w", id, err)
	}

	t.Payload = decoded

	//art-dupl:accept dialect twin of queue/sqlite Get decode tail; only the driver surface differs
	return t, nil
}

// listWhere builds the shared WHERE clause for List and CountTasks.
func listWhere(f queue.Filter) (string, []any) {
	where := []string{"1=1"}
	args := []any{}

	if f.Project != nil {
		where = append(where, fmt.Sprintf("project = $%d", len(args)+1))
		args = append(args, *f.Project)
	}

	if f.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, string(*f.Status))
	}

	if f.Type != nil {
		where = append(where, fmt.Sprintf("type = $%d", len(args)+1))
		args = append(args, *f.Type)
	}

	if f.Since != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, f.Since.UnixMilli())
	}

	if f.Parked != nil && *f.Parked {
		where = append(where, fmt.Sprintf("status = 'pending' AND not_before > $%d", len(args)+1))
		args = append(args, time.Now().UnixMilli())
	}

	if f.PriorityMin != nil {
		where = append(where, fmt.Sprintf("priority >= $%d", len(args)+1))
		args = append(args, *f.PriorityMin)
	}

	if f.PriorityMax != nil {
		where = append(where, fmt.Sprintf("priority <= $%d", len(args)+1))
		args = append(args, *f.PriorityMax)
	}

	if f.Query != "" {
		like := "%" + escapeLike(strings.ToLower(f.Query)) + "%"
		next := len(args)

		// Six searchable columns, one bind each: id, type, project,
		// payload, lease_owner, last_error.
		const likeColumns = 6

		binds := make([]string, likeColumns)
		for i := range binds {
			binds[i] = fmt.Sprintf("$%d", next+1+i)
		}

		where = append(where, fmt.Sprintf(
			`(id ILIKE %s OR type ILIKE %s OR
			project ILIKE %s OR payload ILIKE %s OR
			lease_owner ILIKE %s OR last_error ILIKE %s)`,
			binds[0], binds[1], binds[2], binds[3], binds[4], binds[5],
		))

		for range likeColumns {
			args = append(args, like)
		}
	}

	//art-dupl:accept dialect twin of queue/sqlite listWhere tail; $N vs ? placeholders only
	return strings.Join(where, " AND "), args
}

// List returns tasks matching the filter, ordered by stored priority
// descending then creation age ascending.
func (s *Store[T]) List(ctx context.Context, f queue.Filter) ([]task.Task[T], error) {
	where, args := listWhere(f)

	q := `SELECT ` + taskColumns + ` FROM tasks WHERE ` + where + `
	      ORDER BY priority DESC, created_at ASC`

	if f.Limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, f.Limit)
	}

	if f.Offset > 0 {
		q += fmt.Sprintf(` OFFSET %d`, f.Offset)
	}

	rows, err := s.pool.Query(ctx, q, args...)
	//art-dupl:accept dialect twin of queue/sqlite List query prologue
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []task.Task[T]

	for rows.Next() {
		t, err := s.scanTask(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, t)
	}

	//art-dupl:accept dialect twin of queue/sqlite List scan loop
	return out, rows.Err()
}

// CountTasks counts the tasks matching the filter.
func (s *Store[T]) CountTasks(ctx context.Context, f queue.Filter) (int, error) {
	where, args := listWhere(f)

	var n int

	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE `+where, args...).Scan(&n)

	return n, err
}

// StatusCounts counts tasks per status in one GROUP BY.
// art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) StatusCounts(ctx context.Context) (map[task.Status]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	//art-dupl:accept dialect twin of queue/sqlite StatusCounts scan loop
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	//art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
	out := make(map[task.Status]int)

	for rows.Next() {
		var (
			status task.Status
			n      int
		)

		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}

		out[status] = n
	}

	return out, rows.Err()
}

// escapeLike escapes LIKE wildcards so a query containing %, _ or \
// matches literally.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)

	return s
}
