// Package readmodel provides a typed read-model store over an untyped key-value
// [Backend].
//
// The deployer supplies a [Backend] (which is an alias for [kv.Store]); the
// application wraps it with a typed [Store] parameterised by its value type T
// and key type K. This keeps infrastructure decisions out of application code:
// the deployer chooses where data lives (memory, Pebble, SQL), the application
// only ever imports readmodel.
//
// # Why a typed wrapper?
//
// [kv.Store] is deliberately untyped ([]byte in, []byte out). That is correct
// for a layer-0 abstraction, but application code wants type safety: a [Store]
// that holds [Todo] values addressed by [TodoID] keys, with serialization
// handled once. [Store] provides exactly that, with no new interface to
// implement — [kv.MemStore] and [pebble.KVAdapter] already satisfy [Backend].
//
// # Key types
//
// K must implement [fmt.Stringer]. All branded identifiers from the id package
// ([id.Of]) already do, so [Store] works with them out of the box:
//
//	type TodoMarker struct{}
//	type TodoID = id.Of[TodoMarker]
//
//	store := readmodel.New[Todo, TodoID](backend)
//	got, err := store.Get(ctx, id)
//
// For plain string keys, declare a one-line named type with a String method:
//
//	type EmailKey string
//	func (e EmailKey) String() string { return string(e) }
//
// Use [WithKeyFunc] to override the default key encoding (e.g. raw bytes
// instead of the string form).
//
// # Namespacing
//
// When multiple read models share one [Backend], use [WithKeyPrefix] to give
// each its own keyspace:
//
//	todos := readmodel.New[Todo, TodoID](backend, readmodel.WithKeyPrefix("todos:"))
//	users := readmodel.New[User, UserID](backend, readmodel.WithKeyPrefix("users:"))
package readmodel
