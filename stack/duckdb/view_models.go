package duckdb

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SQLViewModel creates a [storage.SQLViewStore] from the Bundle's DuckDB
// database handle and the given column mapper. The store gets its own table
// with real, queryable columns — enabling server-side WHERE, ORDER BY, and
// pagination that KV-backed read models cannot do.
//
// DuckDB excels at analytical queries on these view tables: GROUP BY
// aggregations, window functions, and columnar scans are all native. This
// makes it ideal for dashboard read models and reporting projections.
//
// Call it as:
//
//	bundle, _ := duckdb.New("analytics.db")
//	mapper := storage.ViewMapper[UserView]{
//	    Table: "users_view",
//	    Columns: []storage.ViewColumn[UserView]{
//	        {Name: "name", Type: "VARCHAR", Extract: func(v *UserView) any { return v.Name }},
//	        {Name: "email", Type: "VARCHAR", Extract: func(v *UserView) any { return v.Email }},
//	    },
//	    ScanRow: func(scan func(dest ...any) error) (*UserView, error) { ... },
//	}
//	store, _ := duckdb.SQLViewModel[UserView, UserID](bundle, mapper)
//	mat := stack.Materialize[UserView, UserID]{Store: store, ...}
//
// The table is auto-created on first call.
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	mapper storage.ViewMapper[V],
) (*storage.SQLViewStore[V, K], error) {
	return sqlopt.SQLViewModel[V, K](b, sqlpkg.DuckDBDialect{}, mapper, "duckdb")
}
