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
//
// All of these are removed in v5 (ADR-0123): metaengine engines with layout
// planning replace SQL view stores. See the storage/view package documentation.
type (
	// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
	ViewColumn[V any] = view.ViewColumn[V]
	// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
	ViewMapper[V any] = view.ViewMapper[V]
	// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
	IndexSpec = view.IndexSpec
	// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
	SQLViewStore[V any, K fmt.Stringer] = view.SQLViewStore[V, K]
	// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
	ViewStoreOption = view.ViewStoreOption
)

// NewSQLiteViewStore is re-exported for backward compatibility.
//
// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
func NewSQLiteViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewSQLiteViewStore[V, K](db, mapper, opts...)
}

// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
func NewSQLViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewSQLViewStore[V, K](db, mapper, opts...)
}

// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
func NewViewStoreWithDialect[V any, K fmt.Stringer](
	db *sql.DB,
	dialect sqlpkg.Dialect,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return view.NewViewStoreWithDialect[V, K](db, dialect, mapper, opts...)
}

// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
func AutoMapper[V any](table string) ViewMapper[V] {
	return view.AutoMapper[V](table)
}

// Deprecated: removed in v5 (ADR-0123): see the storage/view package documentation.
func AutoMapperWithTombstone[V any](table, tombstoneCol string) ViewMapper[V] {
	return view.AutoMapperWithTombstone[V](table, tombstoneCol)
}

func WithoutViewAutoMigrate() ViewStoreOption {
	return view.WithoutViewAutoMigrate()
}
