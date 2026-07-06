package turso

import (
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// ErrNoDatabase is returned when the Bundle was not created by an SQL preset
// and therefore has no *sql.DB handle for SQLViewModel.
var ErrNoDatabase = errorfamily.NewRejection("turso.no_database",
	"turso: bundle has no SQL database handle")

// SQLViewModel creates a [storage.SQLViewStore] from the Bundle's Turso
// database handle and the given column mapper. The store gets its own table
// with real, queryable columns — enabling server-side WHERE, ORDER BY, and
// pagination that KV-backed read models cannot do.
//
// Call it as:
//
//	bundle, _ := turso.New("app.db")
//	mapper := storage.ViewMapper[UserView]{
//	    Table: "users_view",
//	    Columns: []storage.ViewColumn[UserView]{
//	        {Name: "name", Type: "TEXT", Extract: func(v *UserView) any { return v.Name }},
//	        {Name: "email", Type: "TEXT", Extract: func(v *UserView) any { return v.Email }},
//	    },
//	    ScanRow: func(scan func(dest ...any) error) (*UserView, error) { ... },
//	}
//	store, _ := turso.SQLViewModel[UserView, UserID](bundle, mapper)
//	mat := stack.Materialize[UserView, UserID]{Store: store, ...}
//
// The table is auto-created on first call. Pass [storage.WithoutViewAutoMigrate]
// via [storage.NewSQLiteViewStore] directly if you manage schemas yourself.
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	mapper storage.ViewMapper[V],
) (*storage.SQLViewStore[V, K], error) {
	db, ok := b.Database().(*sql.DB)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	store, err := storage.NewSQLiteViewStore[V, K](db, mapper)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "turso.create_view_model",
			"create view model")
	}

	return store, nil
}
