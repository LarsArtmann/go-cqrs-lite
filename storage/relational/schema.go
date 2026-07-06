package relational

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RelationalSchema declares the set of SQL tables a relational projection owns
// and writes to. Unlike [ViewMapper] (which maps a single view type to a single
// table), RelationalSchema describes a full relational read model: multiple
// related tables, foreign keys, junction tables, and history tables.
//
// The schema auto-migrates dialect-independently (SQLite and PostgreSQL both
// accept the generated CREATE TABLE IF NOT EXISTS statements and the common
// column types TEXT, INTEGER, REAL, BLOB). The backend is chosen at deployment
// time via the [sql.Dialect], not at development time — so projection handlers
// written against this schema are portable across SQLite and PostgreSQL.
type RelationalSchema struct {
	Tables []RelationalTable
}

// Table returns the named table definition, or nil if no such table is declared.
func (s RelationalSchema) Table(name string) *RelationalTable {
	for i := range s.Tables {
		if s.Tables[i].Name == name {
			return &s.Tables[i]
		}
	}

	return nil
}

// Validate checks the schema for structural errors: duplicate table names,
// empty table names, columns without names, and primary-key columns that do
// not exist in the table.
func (s RelationalSchema) Validate() error {
	if len(s.Tables) == 0 {
		return errSchemaNoTables
	}

	seen := make(map[string]struct{}, len(s.Tables))

	for i := range s.Tables {
		t := s.Tables[i]

		if err := t.validate(); err != nil {
			return errorfamily.WrapRejection(err,
				"relational.schema_table", fmt.Sprintf("table %q", t.Name))
		}

		if _, dup := seen[t.Name]; dup {
			return errorfamily.WrapRejection(errSchemaDuplicateTable,
				"relational.schema_duplicate_table",
				fmt.Sprintf("duplicate table %q", t.Name))
		}

		seen[t.Name] = struct{}{}
	}

	return nil
}

// RelationalTable describes one SQL table in a [RelationalSchema].
//
// Columns lists the data columns. PrimaryKey optionally names the columns that
// form the primary key (e.g. []string{"guild_id","user_id","role_id"} for a
// junction table). When empty, no PRIMARY KEY clause is emitted — use this for
// tables whose key is an auto-incrementing column declared in its own Type
// (e.g. {Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT"}).
type RelationalTable struct {
	Name       string
	Columns    []RelationalColumn
	PrimaryKey []string
}

func (t RelationalTable) validate() error {
	if t.Name == "" {
		return errSchemaTableNoName
	}

	if len(t.Columns) == 0 {
		return errSchemaTableNoColumns
	}

	colNames := make(map[string]struct{}, len(t.Columns))

	for i := range t.Columns {
		c := t.Columns[i]

		if c.Name == "" {
			return errorfamily.WrapRejection(errSchemaColumnNoName,
				"relational.schema_column_no_name",
				fmt.Sprintf("column %d", i))
		}

		if c.Type == "" {
			return errorfamily.WrapRejection(errSchemaColumnNoType,
				"relational.schema_column_no_type",
				fmt.Sprintf("column %q", c.Name))
		}

		if _, dup := colNames[c.Name]; dup {
			return errorfamily.WrapRejection(errSchemaDuplicateColumn,
				"relational.schema_duplicate_column",
				fmt.Sprintf("column %q", c.Name))
		}

		colNames[c.Name] = struct{}{}
	}

	for _, pk := range t.PrimaryKey {
		if _, ok := colNames[pk]; !ok {
			return errorfamily.WrapRejection(errSchemaUnknownPKColumn,
				"relational.schema_unknown_pk",
				fmt.Sprintf("primary key column %q", pk))
		}
	}

	return nil
}

// RelationalColumn describes one column in a [RelationalTable].
//
// Type is a portable SQL type declaration ("TEXT", "INTEGER", "REAL", "BLOB",
// or "INTEGER PRIMARY KEY AUTOINCREMENT"). It is emitted verbatim into the
// CREATE TABLE statement. Nullable defaults to false (NOT NULL); set to true
// for columns that may hold NULL.
type RelationalColumn struct {
	Name     string
	Type     string
	Nullable bool
}

// DDL returns the CREATE TABLE IF NOT EXISTS statement for one table.
// The statement is portable across SQLite and PostgreSQL.
func (t RelationalTable) DDL() string {
	parts := make([]string, 0, len(t.Columns)+1)

	for _, c := range t.Columns {
		col := c.Name + " " + c.Type
		if !c.Nullable {
			col += " NOT NULL"
		}

		parts = append(parts, col)
	}

	if len(t.PrimaryKey) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(t.PrimaryKey, ", ")+")")
	}

	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n\t%s\n)",
		t.Name,
		strings.Join(parts, ",\n\t"),
	)
}

// Migrate creates all tables in the schema (CREATE TABLE IF NOT EXISTS).
// It is idempotent and safe to call on every startup.
func (s RelationalSchema) Migrate(ctx context.Context, db *sql.DB) error {
	if err := s.Validate(); err != nil {
		return err
	}

	for _, t := range s.Tables {
		if _, err := db.ExecContext(ctx, t.DDL()); err != nil {
			return errorfamily.WrapTransient(err, "relational.migrate",
				fmt.Sprintf("migrate table %q", t.Name))
		}
	}

	return nil
}
