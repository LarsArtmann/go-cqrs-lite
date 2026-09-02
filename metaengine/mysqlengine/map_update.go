package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapUpdate implements metaengine.MapUpdater: a transactional read-modify-
// write on one key. Planned collections update the planned table, planless
// collections update meta_map. Concurrent updates on the same key serialize
// via SELECT ... FOR UPDATE inside the transaction (InnoDB row locks; the
// MySQL driver pool is multi-writer). Inside an outer RunInTx the active
// transaction participates.
func (e *mysqlEngine) MapUpdate(
	ctx context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	if plan, ok := e.planFor(col); ok {
		return e.mapUpdatePlanned(ctx, plan, key, update)
	}

	return e.updateMetaMap(ctx, col, key, update)
}

// mapUpdatePlanned runs the read-modify-write against the planned table.
func (e *mysqlEngine) mapUpdatePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	update func(prev any) any,
) error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	setFn := func(ctx context.Context, q metaengine.SQLExec, keyStr string, val any) error {
		return execPlannedUpsert(ctx, q, plan, keyStr, val)
	}

	if e.activeTx.Load() != nil {
		return updatePlannedValue(ctx, e.conn(), plan.Table, fmt.Sprint(key), update, setFn)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine.mapUpdatePlanned: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err := updatePlannedValue(ctx, tx, plan.Table, fmt.Sprint(key), update, setFn); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine.mapUpdatePlanned: commit: %w", err)
	}

	return nil
}

// updateMetaMap runs the read-modify-write against meta_map for planless
// collections.
func (e *mysqlEngine) updateMetaMap(
	ctx context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	keyStr := fmt.Sprint(key)

	rm := func(ctx context.Context, q metaengine.SQLExec) error {
		var raw []byte

		err := q.QueryRowContext(
			ctx,
			`SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ? AND `+keyCol+` = ? FOR UPDATE`,
			col,
			keyStr,
			//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
		).Scan(&raw)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("mysqlengine.MapUpdate: read: %w", err)
		}

		var prev any
		if err == nil {
			if uerr := json.Unmarshal(raw, &prev); uerr != nil {
				return fmt.Errorf("mysqlengine.MapUpdate: unmarshal: %w", uerr)
			}
		}

		newVal := update(prev)

		data, merr := json.Marshal(newVal)
		if merr != nil {
			return fmt.Errorf("mysqlengine.MapUpdate: marshal: %w", merr)
		}

		if _, uerr := q.ExecContext(ctx,
			`INSERT INTO meta_map (collection, `+keyCol+`, value)
			 VALUES (?, ?, ?)
			 ON DUPLICATE KEY UPDATE value = VALUES(value)`,
			col, keyStr, string(data),
		); uerr != nil {
			return fmt.Errorf("mysqlengine.MapUpdate: write: %w", uerr)
		}

		return nil
	}

	if e.activeTx.Load() != nil {
		return rm(ctx, e.conn())
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine.MapUpdate: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err := rm(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine.MapUpdate: commit: %w", err)
	}

	return nil
}

// updatePlannedValue is the shared planned-table read-modify-write body:
// SELECT the JSON value (row-locked when q is a dedicated transaction),
// apply update, write back through setFn. A missing key reads as a nil prev
// (create-on-update, the metaengine.MapUpdater contract).
//
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func updatePlannedValue(
	ctx context.Context,
	q metaengine.SQLExec,
	table string,
	keyStr string,
	update func(prev any) any,
	setFn func(ctx context.Context, q metaengine.SQLExec, keyStr string, val any) error,
) error {
	var raw []byte

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	err := q.QueryRowContext(ctx,
		fmt.Sprintf("SELECT CAST(value AS CHAR) FROM %s WHERE %s = ? FOR UPDATE",
			backtickIdent(table), keyCol),
		keyStr,
	).Scan(&raw)
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("mysqlengine.mapUpdatePlanned: read: %w", err)
	}

	var prev any
	if err == nil {
		if uerr := json.Unmarshal(raw, &prev); uerr != nil {
			return fmt.Errorf("mysqlengine.mapUpdatePlanned: unmarshal: %w", uerr)
		}
	}

	return setFn(ctx, q, keyStr, update(prev))
}
