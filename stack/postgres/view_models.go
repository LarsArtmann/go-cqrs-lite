package postgres

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
// errors.Is(err, postgres.ErrNoDatabase) still match because errorfamily.Rejection
// compares error code + family, not pointer identity. The actual error is now
// constructed inside [sqlopt.SQLViewModel].
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
	return sqlopt.SQLViewModel[V, K](b, sqlpkg.PostgresDialect{}, mapper, "postgres")
}
