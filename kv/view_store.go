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
// Conditions are AND-joined into a parameterised WHERE clause — injection-safe
// by construction. Column names go in [Condition.Column] (trusted), user input
// goes only in [Condition.Value] / [Condition.Values].
//
// OrderBy is a column name (default: "key"). Desc reverses the order.
// Limit and Offset control pagination; zero Limit means no limit.
type ViewQuery struct {
	Conditions []Condition
	OrderBy    string
	Desc       bool
	Limit      int
	Offset     int
}

// ViewQuerier is an optional capability implemented by view stores that support
// server-side filtering, ordering, and pagination (e.g. SQL-backed stores).
//
// Stores that only support full-scan iteration (e.g. kv.TypedStore over a KV
// backend) do NOT implement this interface. Consumers should check at runtime:
//
//	if q, ok := store.(kv.ViewQuerier[MyView]); ok {
//	    results, _ := q.Query(ctx, kv.ViewQuery{
//	        Conditions: []kv.Condition{{Column: "active", Op: kv.OpEq, Value: true}},
//	    })
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

// ViewCounter is an optional capability implemented by view stores that can
// count records matching a query without loading them. This is far cheaper
// than calling Scan/Query and checking len() — SQL-backed stores translate it
// to SELECT COUNT(*).
//
// When q.Conditions is empty, all records are counted. When the store also
// implements [TombstoneQuerier], the caller can combine a tombstone filter
// into the Conditions for a tombstone-aware count.
type ViewCounter[V any] interface {
	Count(ctx context.Context, q ViewQuery) (int64, error)
}

// ViewResetter is an optional capability implemented by view stores that can
// remove all records in a single operation. This is used for projection
// resets — wiping a read model before rebuilding it from the event journal.
//
// SQL-backed stores translate this to DELETE FROM table. KV-backed stores
// iterate and delete each key.
type ViewResetter[V any] interface {
	DeleteAll(ctx context.Context) error
}

// ViewBatchSetter is an optional capability implemented by view stores that
// support atomic batch upserts. This is critical for projection replay
// throughput — replaying thousands of events one Set at a time is O(n) round
// trips; BatchSet reduces that to O(n / batchSize).
//
// The batch size limit depends on the backend (SQLite has a 999-parameter
// limit per statement). The implementation chunks automatically.
type ViewBatchSetter[V any, K fmt.Stringer] interface {
	BatchSet(ctx context.Context, items []ViewItem[V, K]) error
}

// ViewItem pairs a key and value for batch operations.
type ViewItem[V any, K fmt.Stringer] struct {
	Key   K
	Value *V
}

// Condition is a single WHERE-clause predicate. Conditions are AND-joined.
//
// For scalar operators (=, !=, <, <=, >, >=, LIKE) set Value. For [OpIn] set
// Values instead.
//
// Example:
//
//	[]kv.Condition{
//	    {Column: "age", Op: kv.OpGte, Value: 18},
//	    {Column: "status", Op: kv.OpIn, Values: []any{"active", "pending"}},
//	}
type Condition struct {
	Column string
	Op     Operator
	Value  any

	// Values holds the set for [OpIn]. Ignored for all other operators.
	Values []any
}

// Operator is a comparison operator for [Condition].
type Operator string

const (
	OpEq   Operator = "="
	OpNeq  Operator = "!="
	OpLt   Operator = "<"
	OpLte  Operator = "<="
	OpGt   Operator = ">"
	OpGte  Operator = ">="
	OpLike Operator = "LIKE"
	OpIn   Operator = "IN"
)

// Compile-time assertion: *TypedStore satisfies ViewStore.
var _ ViewStore[any, dummyStringer] = (*TypedStore[any, dummyStringer])(nil)

type dummyStringer string

func (dummyStringer) String() string { return "" }
