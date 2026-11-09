package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// validateDeps enforces enqueue-time dep existence: every dep ID must
// reference an existing task row. One EXISTS probe per dep (dep lists
// are small fan-ins; MySQL and MariaDB have no shared JSON anti-join
// surface). Collects every missing ID for the error. This is also the
// cycle guard — see queue.ErrDanglingDep.
func validateDeps(ctx context.Context, tx *sql.Tx, deps []task.ID) error {
	if len(deps) == 0 {
		return nil
	}

	var missing []string

	for _, d := range deps {
		var exists bool

		err := tx.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM tasks WHERE id = ?)`, d.String()).Scan(&exists)
		if err != nil {
			return fmt.Errorf("validate deps: %w", err)
		}

		if !exists {
			missing = append(missing, d.String())
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", queue.ErrDanglingDep, strings.Join(missing, ", "))
	}

	return nil
}
