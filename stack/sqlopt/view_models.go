package sqlopt

import (
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SQLViewModel creates a [storage.SQLViewStore] from the Bundle's SQL database
// handle and the given column mapper. It is the shared implementation used by
// the preset-specific constructors ([sqlite.SQLViewModel], [turso.SQLViewModel],
// [postgres.SQLViewModel]) so each preset is a one-line delegate.
//
// dialect selects the SQL flavour (sqlpkg.SQLiteDialect{} or sqlpkg.PostgresDialect{}).
// preset is the lowercase preset name embedded in error codes
// ("sqlite", "turso", "postgres"); it lets each preset surface a recognisable
// error location while sharing the construction logic.
//
// On a missing database handle the returned error is an errorfamily.Rejection
// with code "<preset>.no_database" and family Rejection, so callers can still
// match it via errors.Is against their package-level ErrNoDatabase sentinel
// (Rejection.Is compares code+family, see go-error-family/error.go).
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	dialect sqlpkg.Dialect,
	mapper storage.ViewMapper[V],
	preset string,
) (*storage.SQLViewStore[V, K], error) {
	db, ok := b.Database().(*sql.DB)
	if !ok || db == nil {
		return nil, errorfamily.NewRejection(preset+".no_database",
			preset+": bundle has no SQL database handle")
	}

	store, err := storage.NewViewStoreWithDialect[V, K](db, dialect, mapper)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, preset+".create_view_model",
			"create view model")
	}

	return store, nil
}
