package dgraphengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetNodeTypes are the dgraph.type values the engine stamps on every node
// it owns. ResetEngine deletes exactly these nodes; foreign data in a shared
// Dgraph cluster keeps its types untouched.
var resetNodeTypes = []string{
	"MetaMapEntry",
	"MetaSetEntry",
	"MetaCounterEntry",
	"MultimapEntry",
	"LogEntry",
	"StreamLogEntry",
	"GraphNode",
	"SearchDoc",
}

// ResetEngine implements [metaengine.EngineResetter]: it deletes every
// engine-owned node — all nodes carrying one of the engine's dgraph.type
// values — in ONE upsert, returning the engine to its empty post-construction
// state so a journal replay rebuilds every collection from zero. Predicate
// schemas survive; edge predicates disappear with their GraphNode endpoints.
//
// Sequence monotonicity holds by construction: stream-log and journal
// sequences are UnixNano timestamps (wall clock), so post-reset entries
// always sort after pre-reset ones and a consumer holding a pre-reset
// resumption token never skips replayed entries.
//
// The delete runs as one Raft-proposed upsert (atomic), routed through the
// engine's standard doWrite so contention on the shared dgraph.type conflict
// domain is retried like every other write.
func (e *dgraphEngine) ResetEngine(ctx context.Context) error {
	req := &api.Request{
		Query: resetTypeQuery(resetNodeTypes),
		Mutations: []*api.Mutation{{
			DelNquads: []byte(resetDeleteNQuads(resetNodeTypes)),
		}},
	}

	if _, err := e.doWrite(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.ResetEngine: delete engine nodes: %w", err)
	}

	return nil
}

// resetTypeQuery binds one variable per engine node type.
func resetTypeQuery(types []string) string {
	var b strings.Builder

	b.WriteString("query resetEngine {")

	for i, nodeType := range types {
		fmt.Fprintf(&b, " v%d as var(func: type(%s))", i, nodeType)
	}

	b.WriteString(" }")

	return b.String()
}

// resetDeleteNQuads drops ALL predicates of every bound node, which is how a
// node dies in Dgraph.
func resetDeleteNQuads(types []string) string {
	var b strings.Builder

	for i := range types {
		fmt.Fprintf(&b, "uid(v%d) * * . ", i)
	}

	return b.String()
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*dgraphEngine)(nil)
