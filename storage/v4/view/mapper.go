package view

// ViewColumn describes one SQL column that maps a field of view type V.
//
// Name is the SQL column name. Type is the SQL type declaration ("TEXT",
// "INTEGER", "REAL", etc.). Extract returns the column value from a *V for
// INSERT/UPDATE. Extract must not be nil.
//
// Deprecated: removed in v5 (ADR-0123): see the package documentation.
type ViewColumn[V any] struct {
	Name    string
	Type    string
	Extract func(v *V) any
}

// ViewMapper defines how view type V maps to a dedicated SQL table.
//
// Table is the SQL table name (e.g. "users_view"). Columns lists the data
// columns; the key column (always TEXT PRIMARY KEY) is added automatically and
// must NOT appear in Columns. ScanRow reconstructs a *V from a sql.Rows.Scan
// callback — the dest slice matches the order of Columns exactly.
//
// Example:
//
//	mapper := storage.ViewMapper[TodoView]{
//	    Table: "todos_view",
//	    Columns: []storage.ViewColumn[TodoView]{
//	        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
//	        {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
//	        {Name: "tombstoned", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Tombstoned }},
//	    },
//	    ScanRow: func(scan func(dest ...any) error) (*TodoView, error) {
//	        var v TodoView
//	        if err := scan(&v.Title, &v.Completed, &v.Tombstoned); err != nil {
//	            return nil, err
//	        }
//	        return &v, nil
//	    },
//	    TombstoneColumn: "tombstoned", // enables server-side tombstone filtering
//	}
//
// Deprecated: removed in v5 (ADR-0123): see the package documentation.
type ViewMapper[V any] struct {
	Table   string
	Columns []ViewColumn[V]
	ScanRow func(scan func(dest ...any) error) (*V, error)

	// TombstoneColumn optionally names a boolean/integer column that marks
	// tombstoned records (0 = active, non-zero = tombstoned). When set, the
	// store implements [kv.TombstoneQuerier] and [Materialize.List] pushes
	// tombstone filtering to SQL instead of loading every record.
	//
	// The column must also appear in Columns with its Extract function.
	TombstoneColumn string

	// Indexes optionally declares secondary indexes to create on the table.
	// Each index is created via CREATE INDEX IF NOT EXISTS during auto-migration.
	// Use indexes to accelerate frequently-filtered columns.
	//
	// Example:
	//   Indexes: []storage.IndexSpec{
	//       {Name: "idx_email", Columns: []string{"email"}},
	//       {Name: "idx_age_status", Columns: []string{"age", "status"}},
	//   }
	Indexes []IndexSpec
}

// IndexSpec declares a secondary index on a view table.
//
// Deprecated: removed in v5 (ADR-0123): see the package documentation.
type IndexSpec struct {
	// Name is the SQL index name (must be unique within the database).
	Name string
	// Columns are the indexed column names. Composite indexes list
	// multiple columns in order.
	Columns []string
	// Where optionally adds a partial-index predicate, producing
	// `CREATE INDEX ... WHERE <where>`. Empty means a full (non-partial)
	// index. Use partial indexes to keep frequently-filtered subsets small
	// (e.g. `WHERE deleted_at IS NULL`). The caller is responsible for the
	// SQL syntax — it is appended verbatim, no parameterisation.
	Where string
}
