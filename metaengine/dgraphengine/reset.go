package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetTypePredicates maps every engine dgraph.type to its full predicate
// set. ResetEngine nulls each of these predicates (plus dgraph.type) on every
// node of that type — the wildcard form (`uid(v) * * .`) is silently ignored
// by Dgraph's structured mutation path, so deletes are explicit (the same
// lesson MapDelete's DeleteJson already encodes). When a backend gains a
// predicate, this table AND the reset test must grow together.
var resetTypePredicates = map[string][]string{
	"MetaMapEntry": {"cqrs.map_collection", "cqrs.map_key", "cqrs.map_value"},
	"MetaSetEntry": {"cqrs.set_collection", "cqrs.set_key"},
	"MetaCounterEntry": {
		"cqrs.counter_collection", "cqrs.counter_key", "cqrs.counter_value",
	},
	"MultimapEntry": {
		"cqrs.multimap_collection", "cqrs.multimap_key", "cqrs.multimap_value",
	},
	"LogEntry": {"cqrs.log_collection", "cqrs.log_seq", "cqrs.log_value"},
	"StreamLogEntry": {
		"cqrs.stream_log_collection", "cqrs.stream_log_stream",
		"cqrs.stream_log_seq", "cqrs.stream_log_value",
	},
	"GraphNode": {"cqrs.node_collection", "cqrs.node_id"},
	"SearchDoc": {"cqrs.search_collection", "cqrs.search_id", "cqrs.search_content"},
}

// resetTypeOrder gives the var-binding order a stable iteration sequence.
var resetTypeOrder = []string{
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
// engine-owned node (one upsert nulling all typed predicates) and drops every
// dynamically-created edge predicate (cqrs.edge.*), returning the engine to
// its empty post-construction state so a journal replay rebuilds every
// collection from zero. The fixed predicate schema from init() survives;
// edge predicates are schema-level engine state and are dropped wholesale —
// post-construction, they do not exist until the first GraphAddEdge.
// Foreign data in a shared Dgraph cluster is never touched (no engine
// dgraph.type, no cqrs.edge. prefix).
//
// Sequence monotonicity holds by construction: stream-log and journal
// sequences are UnixNano timestamps (wall clock), so post-reset entries
// always sort after pre-reset ones and a consumer holding a pre-reset
// resumption token never skips replayed entries.
//
// The node delete is one Raft-proposed upsert (atomic) routed through
// doWrite, so contention on the shared dgraph.type conflict domain retries
// like every other write. The subsequent edge-predicate drops are Alters
// (one per predicate) — a failure there leaves nodes already deleted and is
// safe to retry (ResetEngine is idempotent).
func (e *dgraphEngine) ResetEngine(ctx context.Context) error {
	req := &api.Request{
		Query:     resetTypeQuery(resetTypeOrder),
		Mutations: []*api.Mutation{{DelNquads: []byte(resetDeleteNQuads(resetTypeOrder))}},
	}

	if _, err := e.doWrite(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.ResetEngine: delete engine nodes: %w", err)
	}

	if err := e.dropEdgePredicates(ctx); err != nil {
		return fmt.Errorf("dgraphengine.ResetEngine: %w", err)
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

// resetDeleteNQuads nulls every predicate of every bound node. Wildcard VALUE
// deletes (`<pred> *`) are supported; wildcard PREDICATE deletes are not —
// hence the explicit table.
func resetDeleteNQuads(types []string) string {
	var b strings.Builder

	for i, nodeType := range types {
		for _, pred := range resetTypePredicates[nodeType] {
			fmt.Fprintf(&b, "uid(v%d) <%s> * .\n", i, pred)
		}

		fmt.Fprintf(&b, "uid(v%d) <dgraph.type> * .\n", i)
	}

	return b.String()
}

// dropEdgePredicates removes every cqrs.edge.* predicate from the schema.
// Edge predicates are created lazily per graph collection (ensureEdgeSchema);
// dropping them deletes their data and their @reverse index together, which
// is exactly the post-construction state.
func (e *dgraphEngine) dropEdgePredicates(ctx context.Context) error {
	resp, err := e.readTx().Query(ctx, "schema { predicate }")
	if err != nil {
		return fmt.Errorf("read schema predicates: %w", err)
	}

	var parsed struct {
		Schema []struct {
			Predicate string `json:"predicate"`
		} `json:"schema"`
	}

	if err := json.Unmarshal(resp.Json, &parsed); err != nil {
		return fmt.Errorf("decode schema response: %w", err)
	}

	for _, pred := range parsed.Schema {
		if !strings.HasPrefix(pred.Predicate, "cqrs.edge.") {
			continue
		}

		drop := pred.Predicate // loop-var copy for the closure

		err := e.retryOnContention(ctx, false, func() error {
			return e.client.Alter(ctx, &api.Operation{DropAttr: drop})
		})
		if err != nil {
			return fmt.Errorf("drop edge predicate %s: %w", drop, err)
		}
	}

	return nil
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*dgraphEngine)(nil)
