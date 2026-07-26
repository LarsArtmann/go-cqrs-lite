package relational

import (
	"context"
	"fmt"
	"slices"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// resolveCols extracts columns/values from row and fills in default conflict
// columns when none were provided. Shared by UpsertCols and UpsertExpr.
func (s *sqlSink) resolveCols(
	table string,
	row Row,
	conflictCols []string,
) ([]string, []any, []string, error) {
	cols, vals, err := s.rowColumns(table, row)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(conflictCols) == 0 {
		conflictCols = s.conflictTarget(table)
	}

	return cols, vals, conflictCols, nil
}

func (s *sqlSink) Increment(
	ctx context.Context,
	table string,
	key Row,
	counterCol string,
	delta int64,
) error {
	keyCols, keyVals, err := s.rowColumns(table, key)
	if err != nil {
		return err
	}

	if slices.Contains(keyCols, counterCol) {
		return errorfamily.WrapRejection(errSinkCounterInKey,
			"relational.sink_counter_in_key",
			fmt.Sprintf("table %q: counter column %q is also in key", table, counterCol))
	}

	t := s.schema.Table(table)

	counterExists := false

	for _, c := range t.Columns {
		if c.Name == counterCol {
			counterExists = true
			break
		}
	}

	if !counterExists {
		return errorfamily.WrapRejection(errSinkUnknownColumn,
			"relational.sink_unknown_column",
			fmt.Sprintf("table %q: counter column %q", table, counterCol))
	}

	conflictCols := s.conflictTarget(table)

	keyColSet := make(map[string]struct{}, len(keyCols))
	for _, c := range keyCols {
		keyColSet[c] = struct{}{}
	}

	for _, pk := range conflictCols {
		if _, ok := keyColSet[pk]; !ok {
			return errorfamily.WrapRejection(errSinkKeyMissingPK,
				"relational.sink_key_missing_pk",
				fmt.Sprintf("table %q: key missing primary key column %q", table, pk))
		}
	}

	allCols := make([]string, 0, len(keyCols)+1)
	allCols = append(allCols, keyCols...)
	allCols = append(allCols, counterCol)

	allVals := make([]any, 0, len(keyVals)+1)
	allVals = append(allVals, keyVals...)
	allVals = append(allVals, delta)

	pholders := placeholders(s.dialect, len(allCols))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s = COALESCE(%s, 0) + excluded.%s",
		table,
		strings.Join(allCols, ", "),
		pholders,
		strings.Join(conflictCols, ", "),
		counterCol,
		counterCol,
		counterCol,
	)

	if _, err := s.tx.ExecContext(ctx, query, allVals...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_increment",
			fmt.Sprintf("increment %s.%s", table, counterCol))
	}

	return nil
}

func (s *sqlSink) UpsertCols(
	ctx context.Context,
	table string,
	row Row,
	updateCols []string,
	conflictCols ...string,
) error {
	cols, vals, conflictCols, err := s.resolveCols(table, row, conflictCols)
	if err != nil {
		return err
	}

	nonConflict := partitionColumns(cols, conflictCols)

	var targetCols []string
	if len(updateCols) > 0 {
		updateSet := make(map[string]struct{}, len(updateCols))
		for _, c := range updateCols {
			updateSet[c] = struct{}{}
		}

		for _, c := range nonConflict {
			if _, ok := updateSet[c]; ok {
				targetCols = append(targetCols, c)
			}
		}
	}

	setClause := excludedSet(targetCols)
	pholders := placeholders(s.dialect, len(cols))

	onConflict := conflictDoNothing
	if setClause != "" {
		onConflict = "DO UPDATE SET " + setClause
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) %s",
		table, strings.Join(cols, ", "), pholders, strings.Join(conflictCols, ", "), onConflict,
	)

	if _, err := s.tx.ExecContext(ctx, query, vals...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_upsert_cols",
			"upsert cols into "+table)
	}

	return nil
}

func (s *sqlSink) UpsertExpr(
	ctx context.Context,
	table string,
	row Row,
	setExprs []SetExpr,
	conflictCols ...string,
) error {
	cols, vals, conflictCols, err := s.resolveCols(table, row, conflictCols)
	if err != nil {
		return err
	}

	knownCols := s.schema.Table(table)
	if knownCols == nil {
		return errorfamily.WrapRejection(errSinkUnknownTable,
			"relational.sink_unknown_table",
			fmt.Sprintf("table %q", table))
	}

	colSet := make(map[string]struct{}, len(knownCols.Columns))
	for _, c := range knownCols.Columns {
		colSet[c.Name] = struct{}{}
	}

	onConflict := conflictDoNothing
	var args []any
	args = append(args, vals...)

	if len(setExprs) > 0 {
		parts := make([]string, 0, len(setExprs))

		for _, se := range setExprs {
			if _, ok := colSet[se.Column]; !ok {
				return errorfamily.WrapRejection(errSinkUnknownColumn,
					"relational.sink_unknown_column",
					fmt.Sprintf("table %q: SetExpr column %q", table, se.Column))
			}

			parts = append(parts, se.Column+" = "+se.Expr)
			args = append(args, se.Args...)
		}

		onConflict = "DO UPDATE SET " + strings.Join(parts, ", ")
	}

	pholders := placeholders(s.dialect, len(cols))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) %s",
		table, strings.Join(cols, ", "), pholders, strings.Join(conflictCols, ", "), onConflict,
	)

	if _, err := s.tx.ExecContext(ctx, query, args...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_upsert_expr",
			"upsert expr into "+table)
	}

	return nil
}
