package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Planned tables (D2): extracted-column tables per collection, mirroring
// metaengine/pgengine/planned.go. MySQL/MariaDB specifics: `key` is a
// reserved word (backtick-quoted), JSON value column, ON DUPLICATE KEY
// upserts, and ? placeholders. MariaDB's JSON type is LONGTEXT-backed, which
// is fine — extracted columns carry the typed reads.


// backtickIdent quotes a SQL identifier the MySQL/MariaDB way. The shared
// metaengine.QuoteIdent emits SQL-standard double quotes, which default
// MySQL/MariaDB sql_mode rejects for identifiers.
func backtickIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// mysqlPlannedColumn maps a plan's SQLite-ish inferred type to MySQL.
func mysqlPlannedColumn(sqliteType string) string {
	switch strings.ToUpper(sqliteType) {
	case "REAL":
		return "DOUBLE"
	case "INTEGER":
		return "BIGINT"
	default:
		return "TEXT"
	}
}

// mysqlDDL renders the CREATE TABLE + CREATE INDEX statements for a plan in
// the MySQL/MariaDB dialect, as SEPARATE statements: the Go MySQL driver
// rejects multi-statement Exec by default (multiStatements=false).
func mysqlDDL(plan metaengine.LayoutPlan) []string {
	table := backtickIdent(plan.Table)

	stmts := make([]string, 0, 1+len(plan.Indexes))

	var b strings.Builder

	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", table)
	b.WriteString("  `key` VARCHAR(255) PRIMARY KEY,\n")
	b.WriteString("  value JSON NOT NULL")

	for _, c := range plan.Columns {
		fmt.Fprintf(&b, ",\n  %s %s", backtickIdent(c.Name), mysqlPlannedColumn(c.Type))
	}

	b.WriteString("\n)")
	stmts = append(stmts, b.String())

	for _, idx := range plan.Indexes {
		stmts = append(stmts, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
			backtickIdent(idx.Name), table, backtickIdent(idx.Columns[0])))
	}

	return stmts
}

// planFor returns the registered plan for a collection, if any.
func (e *mysqlEngine) planFor(col string) (metaengine.LayoutPlan, bool) {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	plan, ok := e.plans[col]

	return plan, ok
}

// registerPlannedLayout creates the planned table + indexes and stores the
// plan. Called with layoutMu held.
func (e *mysqlEngine) registerPlannedLayout(plan metaengine.LayoutPlan) error {
	for _, stmt := range mysqlDDL(plan) {
		if _, err := e.db.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("mysqlengine.registerPlannedLayout: %w", err)
		}
	}

	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// ApplyLayoutPlan implements metaengine.LayoutPlanApplier: registers a full
// LayoutPlan post-construction and creates the planned table. Conflicting
// re-registrations are rejected.
//
// Note: `CREATE INDEX IF NOT EXISTS` is MariaDB-only syntax; on Oracle MySQL
// planned-table creation for a table that already exists with different
// indexes fails loudly, which is the acceptable fail-loud path (MariaDB is
// the dialect this repo verifies against).
func (e *mysqlEngine) ApplyLayoutPlan(plan metaengine.LayoutPlan) error {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if existing, exists := e.plans[plan.Collection]; exists {
		if !metaengine.PlansColumnCompatible(existing, plan) {
			return fmt.Errorf(
				"%w: collection %q already has columns %v, requested %v",
				metaengine.ErrLayoutConflict,
				plan.Collection,
				existing.ColumnNames(),
				plan.ColumnNames(),
			)
		}

		return nil
	}

	if err := e.registerPlannedLayout(plan); err != nil {
		return fmt.Errorf("mysqlengine.ApplyLayoutPlan: %w", err)
	}

	return nil
}

// mapSetPlanned upserts a key-value pair with extracted columns.
func (e *mysqlEngine) mapSetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("mysqlengine.mapSetPlanned: marshal: %w", err)
	}

	extracted := metaengine.ExtractFields(value, plan.Columns)

	cols := make([]string, 0, 2+len(plan.Columns))
	placeholders := make([]string, 0, 2+len(plan.Columns))
	args := make([]any, 0, 2+len(plan.Columns))

	cols = append(cols, keyCol, "value")
	placeholders = append(placeholders, "?", "?")
	args = append(args, fmt.Sprint(key), string(data))

	for _, c := range plan.Columns {
		cols = append(cols, backtickIdent(c.Name))
		placeholders = append(placeholders, "?")
		args = append(args, extracted[c.Name])
	}

	updates := make([]string, 0, 1+len(plan.Columns))
	updates = append(updates, "value = VALUES(value)")
	for _, c := range plan.Columns {
		updates = append(updates,
			fmt.Sprintf("%s = VALUES(%s)", backtickIdent(c.Name), backtickIdent(c.Name)))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		backtickIdent(plan.Table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "),
	)

	if _, err := e.conn().ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mysqlengine.mapSetPlanned: %w", err)
	}

	return nil
}

// mapGetPlanned reads one row from the planned table.
func (e *mysqlEngine) mapGetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) (any, bool, error) {
	var raw []byte

	err := e.conn().QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT CAST(value AS CHAR) FROM %s WHERE %s = ?",
			backtickIdent(plan.Table), keyCol),
		fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("mysqlengine.mapGetPlanned: %w", err)
	}

	var val any
	if err := json.Unmarshal(raw, &val); err != nil {
		return nil, false, fmt.Errorf("mysqlengine.mapGetPlanned: unmarshal: %w", err)
	}

	return val, true, nil
}

// mapDeletePlanned removes one row from the planned table.
func (e *mysqlEngine) mapDeletePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) error {
	_, err := e.conn().ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE %s = ?", backtickIdent(plan.Table), keyCol),
		fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("mysqlengine.mapDeletePlanned: %w", err)
	}

	return nil
}
