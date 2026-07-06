package postgres

import (
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// ErrNoDatabase is returned when the Bundle was not created by an SQL preset
// and therefore has no *sql.DB handle for SQLViewModel.
var ErrNoDatabase = errorfamily.NewRejection("postgres.no_database",
	"postgres preset: bundle has no SQL database handle")

// SQLViewModel creates a [storage.SQLViewStore] from the Bundle's PostgreSQL
// database handle and the given column mapper. The store gets its own table
// with real, queryable columns — enabling server-side WHERE, ORDER BY, and
// pagination that KV-backed read models cannot do.
//
// Call it as:
//
//	bundle, _ := postgres.New(dsn)
//	mapper := storage.ViewMapper[UserView]{ ... }
//	store, _ := postgres.SQLViewModel[UserView, UserID](bundle, mapper)
//	mat := stack.Materialize[UserView, UserID]{Store: store, ...}
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	mapper storage.ViewMapper[V],
) (*storage.SQLViewStore[V, K], error) {
	db, ok := b.Database().(*sql.DB)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	store, err := storage.NewSQLViewStore[V, K](db, mapper)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "postgres.create_view_model",
			"create view model")
	}

	return store, nil
}
