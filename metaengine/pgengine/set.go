package pgengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// --- metaengine.SetBackend ---
//
// meta_set mirrors sqliteengine's layout (collection, key) with the PK
// covering both columns, so SetContains is an indexed point read. Parity with
// the sqlite/memory set semantics is asserted by the adttest matrix legs that
// activate automatically once the engine implements the backend.

// SetAdd records membership. Re-adding an existing member is a no-op
// (ON CONFLICT DO NOTHING), matching sqlite's INSERT OR IGNORE.
func (e *pgEngine) SetAdd(ctx context.Context, col string, key any) error {
	_, err := e.conn(ctx).ExecContext(
		ctx,
		`INSERT INTO meta_set (collection, key) VALUES ($1, $2)
		 ON CONFLICT (collection, key) DO NOTHING`,
		col, fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("pgengine.SetAdd: %w", err)
	}

	return nil
}

// SetContains reports membership; a missing row is (false, nil).
func (e *pgEngine) SetContains(ctx context.Context, col string, key any) (bool, error) {
	var one int

	err := e.conn(ctx).QueryRowContext(
		ctx,
		`SELECT 1 FROM meta_set WHERE collection = $1 AND key = $2`,
		col, fmt.Sprint(key),
	).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("pgengine.SetContains: %w", err)
	}

	return true, nil
}
