package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sort"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// keyCol is the backtick-escaped `key` column name (reserved in MySQL).
const keyCol = "`key`"

// --- MapBackend ---

func (e *mysqlEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("mysqlengine.MapSet: marshal: %w", err)
	}

	_, err = e.conn().ExecContext(
		ctx,
		`INSERT INTO meta_map (collection, `+keyCol+`, value)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		col, fmt.Sprint(key), string(data),
	)
	if err != nil {
		return fmt.Errorf("mysqlengine.MapSet: %w", err)
	}

	return nil
}

func (e *mysqlEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	var raw []byte

	err := e.conn().QueryRowContext(
		ctx,
		`SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ? AND `+keyCol+` = ?`,
		col, fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("mysqlengine.MapGet: %w", err)
	}

	var val any

	if err := json.Unmarshal(raw, &val); err != nil {
		return nil, false, fmt.Errorf("mysqlengine.MapGet: unmarshal: %w", err)
	}

	return val, true, nil
}

func (e *mysqlEngine) MapDelete(ctx context.Context, col string, key any) error {
	_, err := e.conn().ExecContext(
		ctx,
		`DELETE FROM meta_map WHERE collection = ? AND `+keyCol+` = ?`,
		col, fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("mysqlengine.MapDelete: %w", err)
	}

	return nil
}

// --- CounterBackend ---

func (e *mysqlEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	if len(deltas) == 0 {
		return nil
	}

	keys := make([]string, 0, len(deltas))
	for key := range deltas {
		keys = append(keys, key)
	}

	sort.Strings(keys) // deterministic placeholder ordering

	placeholders := make([]string, len(keys))
	args := make([]any, 0, len(keys)*3)

	for i, key := range keys {
		placeholders[i] = "(?, ?, ?)"
		args = append(args, col, key, deltas[key])
	}

	query := fmt.Sprintf(
		`INSERT INTO meta_counter (collection, `+keyCol+`, value) VALUES %s
		 ON DUPLICATE KEY UPDATE value = value + VALUES(value)`,
		strings.Join(placeholders, ", "),
	)

	if _, err := e.conn().ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mysqlengine.CounterIncrement: %w", err)
	}

	return nil
}

func (e *mysqlEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT `+keyCol+`, value FROM meta_counter WHERE collection = ?`,
		col,
	)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.CounterGet: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	result := make(map[string]int64)

	for rows.Next() {
		var key string

		var val int64

		if err := rows.Scan(&key, &val); err != nil {
			return nil, fmt.Errorf("mysqlengine.CounterGet: scan: %w", err)
		}

		result[key] = val
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("CounterGet: %w", err)
	}

	return result, nil
}
