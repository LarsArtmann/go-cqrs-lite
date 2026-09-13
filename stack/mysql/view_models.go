package mysql

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// ErrNoDatabase is returned when the bundle has no SQL database handle.
var ErrNoDatabase = errorfamily.NewRejection("mysql.no_database",
	"mysql: bundle has no SQL database handle")

// SQLViewModel creates a [storage.SQLViewStore] backed by the MySQL database
// in the bundle, using the MySQL dialect.
func SQLViewModel[V any, K fmt.Stringer](
	b *stack.Bundle,
	mapper storage.ViewMapper[V],
) (*storage.SQLViewStore[V, K], error) {
	return sqlopt.SQLViewModel[V, K](b, sqlpkg.MySQLDialect{}, mapper, "mysql")
}
