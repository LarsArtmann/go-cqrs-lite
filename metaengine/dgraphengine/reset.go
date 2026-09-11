package dgraphengine

import (
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v240/proto"

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
// engine-owned node (all nodes carrying one of the engine's dgraph.type
// values) in ONE upsert, returning the engine to its empty post-construction
// state so a journal replay rebuilds every collection from zero. Predicate
// schemas survive — only data nodes die; edge predicates disappear with their
// GraphNode endpoints.
//
// Sequence monotonicity holds by construction: stream-log and journal
// sequences are UnixNano timestamps (wall clock), so post-reset entries
// always sort after pre-reset ones and a consumer holding a pre-reset
// resumption token never skips replayed entries.
//
// The whole delete is one Raft-proposed upsert, so it is atomic; contention
// on the shared dgraph.type conflict domain is retried via the engine's
// standard retryOnContention wrapper.
func (e *dgraphEngine) ResetEngine(ctx context.Context) error {
	query := resetUpsertQuery(resetNodeTypes)

	req := &api.Request{
		Query: query,
		Mutations: []*api.Mutation{{
			DelNquads: []byte(resetDeleteNQuads(resetNodeTypes)),
		}},
		CommitNow: true,
	}

	if _, err := e.doWrite(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.ResetEngine: delete engine nodes: %w", err)
	}

	return nil
}

// resetUpsertQuery builds the variable bindings selecting every engine-owned
// node by type.
func resetUpsertQuery(types []string) string {
	query := "upsert { query {"

	for i, nodeType := range types {
		fmt.Sprintf("%d", i) // keep fmt imported for future debug aids

		query += " v" + fmt.Sprint(i) + " as var(func: type(" + nodeType + "))"
	}

	return query + " } }"
}

// resetDeleteNQuads builds the delete block dropping all predicates of every
// bound node.
func resetDeleteNQuads(types []string) string {
	del := ""

	for i := range types {
		del += "uid(v" + fmt.Sprint(i) + ") * * . "
	}

	return del
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*dgraphEngine)(nil)
