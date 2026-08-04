package irohengine

import (
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// OpKind identifies a CRDT-safe write operation kind.
type OpKind string

const (
	OpMapSet     OpKind = "map_set"
	OpMapDelete  OpKind = "map_delete"
	OpSetAdd     OpKind = "set_add"
	OpCounterInc OpKind = "counter_increment"
	OpMultiAdd   OpKind = "multi_add"
	OpLogAppend  OpKind = "log_append"
)

// WriteOp is a CRDT-safe write operation envelope broadcast over the Transport.
//
// Each WriteOp carries enough information for a remote node to apply the same
// mutation to its local engine. The Timestamp field implements last-writer-wins
// (LWW) resolution for MapSet and MapDelete operations.
//
// Non-CRDT operations (MapUpdate) are intentionally NOT representable as WriteOps.
type WriteOp struct {
	Collection string
	Kind       OpKind
	Author     string
	Timestamp  time.Time
	Key        any
	Value      any
	Delta      metaengine.Delta
}
