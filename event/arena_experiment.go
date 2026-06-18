//go:build goexperiment.arenas

package event

import "arena"

// ArenaBatchBuilder allocates a batch of events using Go's experimental
// arena allocator. All events allocated from the same arena can be freed
// in one operation, reducing GC pressure in hot paths.
//
// Enable with: go build -tags goexperiment.arenas
// Requires: GOEXPERIMENT=arenas in the Go toolchain.
//
// Usage:
//
//	a := arena.NewArena[Event](100)
//	defer a.Free()
//	builder := NewArenaBatchBuilder(a)
//	evt := builder.NewEvent(...)
//
// Note: arena allocation is experimental and may be removed in future Go versions.
// See ADR-0026 for details.
type ArenaBatchBuilder struct {
	pool *arena.Arena
}

// NewArenaBatchBuilder creates a batch builder backed by the given arena.
func NewArenaBatchBuilder(a *arena.Arena) ArenaBatchBuilder {
	return ArenaBatchBuilder{pool: a}
}

// NewImmutableEvent allocates an ImmutableEvent within the arena.
func (b ArenaBatchBuilder) NewImmutableEvent() *ImmutableEvent {
	return arena.New[ImmutableEvent](b.pool)
}
