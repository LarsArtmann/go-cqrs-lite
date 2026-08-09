package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"math/big"
)

// DecodeFloat converts a database driver scan value to float64.
//
// SQL engines return different Go types for numeric aggregate results:
//   - float64 for SUM/AVG over DOUBLE columns
//   - *big.Int for HUGEINT (DuckDB SUM over INTEGER columns)
//   - int64 for COUNT results
//   - nil for empty sets (COUNT of zero rows returns nil, not 0)
//   - []byte for some driver encodings
//
// This function normalizes all of these to float64, which is what the
// aggregate reader interfaces return. It is used by the DuckDB, SQLite,
// and Postgres engine implementations.
func DecodeFloat(raw any) (float64, error) {
	if raw == nil {
		return 0, nil
	}

	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case *big.Int:
		f, _ := v.Float64()
		return f, nil
	case []byte:
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			return 0, fmt.Errorf("metaengine DecodeFloat: %w", err)
		}

		return f, nil
	default:
		return 0, fmt.Errorf("metaengine DecodeFloat: unexpected type %T", raw)
	}
}

// DecodeFloatResults decodes a slice of raw scan values into a result map
// keyed by each spec's alias. It is the shared result-building step for
// MultiAggregate implementations across DuckDB, SQLite, and Postgres.
func DecodeFloatResults(
	raws []any,
	specs []AggregateSpec,
	errPrefix string,
) (map[string]float64, error) {
	if len(raws) < len(specs) {
		return nil, fmt.Errorf(
			"%s: raw values (%d) fewer than specs (%d)",
			errPrefix,
			len(raws),
			len(specs),
		)
	}
	result := make(map[string]float64, len(specs))
	for i, s := range specs {
		val, err := DecodeFloat(raws[i])
		if err != nil {
			return nil, fmt.Errorf("%s alias %q: %w", errPrefix, s.AliasOr(), err)
		}

		result[s.AliasOr()] = val
	}

	return result, nil
}

// SQLExec is the common interface between *sql.DB and *sql.Tx. Every SQL
// engine implementation (DuckDB, SQLite, Postgres) uses this to route
// operations through the active transaction when one exists. Both *sql.DB
// and *sql.Tx satisfy this interface.
type SQLExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ScanDistinctValues executes a single-column query and collects each row into
// []any. Shared by DuckDB and SQLite engine implementations for DistinctValues.
// The label is used as the error prefix (e.g. "duckdbengine.DistinctValues").
func ScanDistinctValues(
	ctx context.Context,
	q SQLExec,
	query string,
	args []any,
	label string,
) ([]any, error) {
	rows, err := q.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}

	defer DeferClose(rows)

	var result []any

	for rows.Next() {
		var raw any

		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", label, err)
		}

		result = append(result, raw)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}

	return result, nil
}

// MultiAggregateScan executes a single-row aggregate query and decodes the
// results into a map keyed by each spec's alias. Shared by DuckDB, SQLite,
// and Postgres engine implementations for MultiAggregate. The label is used
// as the error prefix (e.g. "duckdbengine.MultiAggregate").
func MultiAggregateScan(
	ctx context.Context,
	q SQLExec,
	query string,
	args []any,
	specs []AggregateSpec,
	label string,
) (map[string]float64, error) {
	raws := make([]any, len(specs))
	ptrs := make([]any, len(specs))

	for i := range raws {
		ptrs[i] = &raws[i]
	}

	if err := q.QueryRowContext(ctx, query, args...).Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}

	return DecodeFloatResults(raws, specs, label)
}
