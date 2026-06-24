package kv

import (
	"context"
	"fmt"
)

// ViewStore is the typed read-model interface that [TypedStore] implements.
//
// It decouples consumers (such as stack.Materialize) from the concrete
// *TypedStore, allowing alternative implementations — notably SQL-backed view
// stores with real columns — to be used interchangeably. Every [TypedStore]
// satisfies this interface; no adapter is needed.
//
// The interface is deliberately minimal: Get, Set, Delete, and Scan. Stores
// that support richer querying (WHERE, ORDER BY, LIMIT) additionally implement
// [ViewQuerier].
type ViewStore[V any, K fmt.Stringer] interface {
	Get(ctx context.Context, key K) (*V, error)
	Set(ctx context.Context, key K, val *V) error
	Delete(ctx context.Context, key K) error
	Scan(ctx context.Context, prefix []byte) ([]*V, error)
}

// ViewQuery describes a filtered, ordered, paginated query against a view store.
//
// Where is a raw SQL WHERE-clause fragment (without the "WHERE" keyword) using
// dialect-appropriate placeholders (? for SQLite, $N for Postgres). Args are
// bound positionally. The caller is responsible for SQL-injection safety: only
// pass column names and operators in Where, never user input.
//
// OrderBy is a column name (default: "key"). Desc reverses the order.
// Limit and Offset control pagination; zero Limit means no limit.
type ViewQuery struct {
	Where   string
	Args    []any
	OrderBy string
	Desc    bool
	Limit   int
	Offset  int
}

// ViewQuerier is an optional capability implemented by view stores that support
// server-side filtering, ordering, and pagination (e.g. SQL-backed stores).
//
// Stores that only support full-scan iteration (e.g. kv.TypedStore over a KV
// backend) do NOT implement this interface. Consumers should check at runtime:
//
//	if q, ok := store.(kv.ViewQuerier[MyView]); ok {
//	    results, _ := q.Query(ctx, kv.ViewQuery{Where: "active = ?", Args: []any{true}})
//	}
type ViewQuerier[V any] interface {
	Query(ctx context.Context, q ViewQuery) ([]*V, error)
}

// TombstoneQuerier is an optional capability implemented by view stores that
// can filter tombstoned records server-side, avoiding a full-table load.
//
// excludeTombstoned and onlyTombstoned are mutually exclusive. When both are
// false, all records are returned (equivalent to IncludeTombstoned).
//
// SQL-backed stores implement this when a tombstone column is configured in the
// ViewMapper. KV-backed stores do not — they fall back to in-memory filtering.
type TombstoneQuerier[V any] interface {
	QueryByTombstone(ctx context.Context, excludeTombstoned, onlyTombstoned bool) ([]*V, error)
}

// Compile-time assertion: *TypedStore satisfies ViewStore.
var _ ViewStore[any, dummyStringer] = (*TypedStore[any, dummyStringer])(nil)

type dummyStringer string

func (dummyStringer) String() string { return "" }
