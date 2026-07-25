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
	Indexes    []IndexSpec
	Uniques    []UniqueSpec
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

	for _, idx := range t.Indexes {
		if idx.Name == "" {
			return errorfamily.WrapRejection(errSchemaIndexNoName,
				"relational.schema_index_no_name",
				fmt.Sprintf("table %q", t.Name))
		}

		for _, ic := range idx.Columns {
			if _, ok := colNames[ic]; !ok {
				return errorfamily.WrapRejection(errSchemaUnknownIndexColumn,
					"relational.schema_unknown_index_col",
					fmt.Sprintf("table %q: index %q column %q", t.Name, idx.Name, ic))
			}
		}
	}

	for _, uq := range t.Uniques {
		if uq.Name == "" {
			return errorfamily.WrapRejection(errSchemaUniqueNoName,
				"relational.schema_unique_no_name",
				fmt.Sprintf("table %q", t.Name))
		}

		for _, uc := range uq.Columns {
			if _, ok := colNames[uc]; !ok {
				return errorfamily.WrapRejection(errSchemaUnknownUniqueColumn,
					"relational.schema_unknown_unique_col",
					fmt.Sprintf("table %q: unique %q column %q", t.Name, uq.Name, uc))
			}
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
//
// Default, when non-empty, emits a DEFAULT clause. The value is a raw SQL
// expression — it is appended verbatim, so callers are responsible for quoting
// literal values (e.g. Default: "'unknown'" for a string literal, Default: "0"
// for an integer, Default: "CURRENT_TIMESTAMP" for a timestamp).
//
// Unique, when true, emits a single-column UNIQUE constraint.
//
// References, when non-empty, emits a REFERENCES clause for a foreign key.
// The value is the full references clause body: "other_table(id)". It is
// appended verbatim after "REFERENCES ", so callers control the target table,
// column, and any ON DELETE / ON UPDATE actions (e.g.
// References: "guilds(id) ON DELETE CASCADE").
type RelationalColumn struct {
	Name       string
	Type       string
	Nullable   bool
	Default    string
	Unique     bool
	References string
}

// IndexSpec declares a secondary index on a [RelationalTable]. It generates a
// portable "CREATE INDEX IF NOT EXISTS" statement during [RelationalSchema.Migrate].
//
// Columns lists the indexed column names (composite indexes use multiple
// entries in order). Where, when non-empty, appends a partial-index predicate
// verbatim ("WHERE deleted_at IS NULL"). The caller is responsible for the
// SQL syntax of Where — it is not parameterised.
type IndexSpec struct {
	Name    string
	Columns []string
	Where   string
}

// UniqueSpec declares a composite UNIQUE constraint on a [RelationalTable].
// Unlike the single-column [RelationalColumn.Unique] flag, UniqueSpec covers
// multi-column uniqueness (e.g. "one reaction per user per message per emoji").
// It emits a "UNIQUE (<cols>)" clause inside the CREATE TABLE statement.
type UniqueSpec struct {
	Name    string
	Columns []string
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

		if c.Default != "" {
			col += " DEFAULT " + c.Default
		}

		if c.Unique {
			col += " UNIQUE"
		}

		if c.References != "" {
			col += " REFERENCES " + c.References
		}

		parts = append(parts, col)
	}

	if len(t.PrimaryKey) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(t.PrimaryKey, ", ")+")")
	}

	for _, uq := range t.Uniques {
		parts = append(parts, "UNIQUE ("+strings.Join(uq.Columns, ", ")+")")
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

		for _, idx := range t.Indexes {
			stmt := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				idx.Name, t.Name, strings.Join(idx.Columns, ", "),
			)

			if idx.Where != "" {
				stmt += " WHERE " + idx.Where
			}

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return errorfamily.WrapTransient(err, "relational.migrate",
					fmt.Sprintf("migrate index %q on %q", idx.Name, t.Name))
			}
		}
	}

	return nil
}
