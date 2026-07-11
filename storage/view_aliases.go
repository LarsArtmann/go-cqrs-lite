package storage

import (
	"database/sql"
	"fmt"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	"github.com/larsartmann/go-cqrs-lite/storage/v4/view"
)

// View sub-package re-exports.
// Consumers importing storage/ can use these types unchanged.
// New consumers should prefer importing storage/view directly.
type (
	ViewColumn[V any]                   = view.ViewColumn[V]
	ViewMapper[V any]                   = view.ViewMapper[V]
	IndexSpec                           = view.IndexSpec
	SQLViewStore[V any, K fmt.Stringer] = view.SQLViewStore[V, K]
	ViewStoreOption                     = view.ViewStoreOption
)

// NewSQLiteViewStore is re-exported for backward compatibility.
func NewSQLiteViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewSQLiteViewStore[V, K](db, mapper, opts...)
}

func NewSQLViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewSQLViewStore[V, K](db, mapper, opts...)
}

func NewViewStoreWithDialect[V any, K fmt.Stringer](
	db *sql.DB,
	dialect sqlpkg.Dialect,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewViewStoreWithDialect[V, K](db, dialect, mapper, opts...)
}

func AutoMapper[V any](table string) ViewMapper[V] {
	return view.AutoMapper[V](table)
}

func AutoMapperWithTombstone[V any](table, tombstoneCol string) ViewMapper[V] {
	return view.AutoMapperWithTombstone[V](table, tombstoneCol)
}

func WithoutViewAutoMigrate() ViewStoreOption {
	return view.WithoutViewAutoMigrate()
}
