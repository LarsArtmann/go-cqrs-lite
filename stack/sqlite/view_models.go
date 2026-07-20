package sqlite

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// ErrNoDatabase is returned when the Bundle was not created by an SQL preset
// and therefore has no *sql.DB handle for SQLViewModel.
//
// Retained as a package-level sentinel for API compatibility: callers using
// errors.Is(err, sqlite.ErrNoDatabase) still match because errorfamily.Rejection
// compares error code + family, not pointer identity. The actual error is now
// constructed inside [sqlopt.SQLViewModel].
var ErrNoDatabase = errorfamily.NewRejection("sqlite.no_database",
	"sqlite: bundle has no SQL database handle")

// SQLViewModel creates a [storage.SQLViewStore] from the Bundle's SQLite
// database handle and the given column mapper. The store gets its own table
// with real, queryable columns — enabling server-side WHERE, ORDER BY, and
// pagination that KV-backed read models cannot do.
//
// Call it as:
//
//	bundle, _ := sqlite.New("app.db")
//	mapper := storage.ViewMapper[UserView]{
//	    Table: "users_view",
//	    Columns: []storage.ViewColumn[UserView]{
//	        {Name: "name", Type: "TEXT", Extract: func(v *UserView) any { return v.Name }},
//	        {Name: "email", Type: "TEXT", Extract: func(v *UserView) any { return v.Email }},
//	    },
//	    ScanRow: func(scan func(dest ...any) error) (*UserView, error) { ... },
//	}
//	store, _ := sqlite.SQLViewModel[UserView, UserID](bundle, mapper)
//	mat := stack.Materialize[UserView, UserID]{Store: store, ...}
//
// The table is auto-created on first call. Pass [storage.WithoutViewAutoMigrate]
// via [storage.NewSQLiteViewStore] directly if you manage schemas yourself.
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	mapper storage.ViewMapper[V],
) (*storage.SQLViewStore[V, K], error) {
	return sqlopt.SQLViewModel[V, K](b, sqlpkg.SQLiteDialect{}, mapper, "sqlite")
}
