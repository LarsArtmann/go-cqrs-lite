package metaengine

import (
	"fmt"
	"sort"
	"strings"
)

// EngineProfile describes what an engine can do and at what cost.
type EngineProfile struct {
	Name     string
	Supports map[ADT]Complexity
}

func (p EngineProfile) SupportsADT(adt ADT) (Complexity, bool) {
	c, ok := p.Supports[adt]

	return c, ok
}

func (p EngineProfile) String() string {
	var parts []string
	for adt, c := range p.Supports {
		parts = append(parts, fmt.Sprintf("%s@%s", adt, c))
	}

	sort.Strings(parts)

	return fmt.Sprintf("%s: %s", p.Name, strings.Join(parts, " "))
}

// Per-ADT backend interfaces (ISP — engines implement only what they support).
// This mirrors the existing kv.ViewStore + ViewQuerier + ViewCounter pattern.

type MapBackend interface {
	MapSet(collection string, key any, value any) error
	MapGet(collection string, key any) (any, bool, error)
	MapDelete(collection string, key any) error
}

// MapUpdater is an optional capability for atomic read-modify-write on map entries.
// Engines that implement this interface handle FoldUpdate without race conditions.
// Mirrors the kv.ViewUpdater pattern from the existing codebase.
type MapUpdater interface {
	MapUpdate(collection string, key any, update func(prev any) any) error
}

// ScanBackend handles filtered+sorted scans for collection queries.
// Filters are runtime predicates (typed closures from FilterOn), not field
// name strings. Sort is a runtime comparator (from SortOn), or nil for default.
type ScanBackend interface {
	MapScan(
		collection string,
		filters []filterPredicate,
		sortFunc func(a, b any) int,
		cursor any,
		limit int,
	) ([]any, error)
}

type SetBackend interface {
	SetAdd(collection string, key any) error
	SetContains(collection string, key any) (bool, error)
}

type CounterBackend interface {
	CounterIncrement(collection string, deltas Delta) error
	CounterGet(collection string) (map[string]int64, error)
}

type GraphBackend interface {
	GraphAddEdge(collection string, edge Edge) error
	GraphNeighbors(collection string, node any, depth int) ([]any, error)
}

// MultimapBackend handles one-to-many key-to-values collections.
type MultimapBackend interface {
	MultiAdd(collection string, key any, value any) error
	MultiGet(collection string, key any) ([]any, error)
}

// LogBackend handles append-only, ordered log collections.
type LogBackend interface {
	LogAppend(collection string, value any) error
	LogTail(collection string, limit int) ([]any, error)
}

// Closer is the lifecycle interface.
type Closer interface {
	Close() error
}

// Engine is a storage backend with a cost profile.
// An engine implements whichever ADT backends it supports.
// The planner checks capabilities at runtime via type assertion.
type Engine interface {
	Profile() EngineProfile
	Closer
}

// Compile-time assertions that memoryEngine implements all backend interfaces.
var (
	_ Engine          = (*memoryEngine)(nil)
	_ MapBackend      = (*memoryEngine)(nil)
	_ MapUpdater      = (*memoryEngine)(nil)
	_ ScanBackend     = (*memoryEngine)(nil)
	_ SetBackend      = (*memoryEngine)(nil)
	_ CounterBackend  = (*memoryEngine)(nil)
	_ GraphBackend    = (*memoryEngine)(nil)
	_ MultimapBackend = (*memoryEngine)(nil)
	_ LogBackend      = (*memoryEngine)(nil)
)
