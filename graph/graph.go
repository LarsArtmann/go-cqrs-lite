// Package graph provides a projection tier for graph-structured read models.
//
// It is the graph counterpart to [storage.RelationalProjection] (relational
// reads) and [stack.Materialize] (document/KV reads): the third fundamental
// data-model tier. Where relational projections flatten events into tables and
// document projections store one blob per key, graph projections merge events
// into nodes and edges — the right shape when the dominant read patterns are
// variable-depth traversal, path-finding, adjacency, or connected-component
// queries (reply chains, social graphs, causation DAGs, role memberships,
// reaction networks).
//
// # Scope: writes are portable, reads are native
//
// The abstraction deliberately stops at the write boundary. Every graph
// database supports MERGE/upsert semantics on nodes and edges (openCypher
// MERGE is spoken by Neo4j, Memgraph, Apache Age, RedisGraph), so a
// [GraphSink] with MergeNode/MergeEdge is genuinely cross-backend — exactly as
// the SQL Dialect tier makes relational writes portable across SQLite/Postgres.
//
// Graph reads are NOT abstracted. Cypher, Gremlin, and GQL differ enough that
// abstracting them is a research problem rather than an engineering one. A
// [GraphDriver] exposes its underlying query mechanism directly; consumers run
// native Cypher/Gremlin against it. This asymmetry is documented, not hidden.
//
// # Backends
//
// The in-memory [MemoryDriver] is the reference implementation: zero
// dependencies, suitable for tests and single-process local use. A Neo4j or
// Memgraph driver lives in a consumer-pulled sibling module (e.g.
// graph/neo4j/) — same convention as storage/pebble and storage/turso being
// separate from storage/.
package graph

// NodeRef identifies a single node by its label and a key property.
//
// A node's identity is the (Label, KeyProp, KeyValue) triple — equivalent to a
// primary key in the relational tier. MergeNode is idempotent against this
// triple: re-projecting the same event merges onto the same node, never
// creates a duplicate.
//
// KeyValue is typed `any` for the same reason database/sql scan targets are:
// node keys may be strings, integers, or branded ID Stringers, and the graph
// driver serialises them according to its native type system. This is storage
// infrastructure, not domain logic — the library-wide no-`any` rule applies to
// domain/business code.
type NodeRef struct {
	Label    string
	KeyProp  string
	KeyValue any
}

// EdgeRef identifies a single directed edge by its type and endpoint NodeRefs.
//
// An edge's identity is (Type, From, To) — there is at most one edge of a given
// type between any ordered pair of nodes. MergeEdge is idempotent against this
// identity; re-projecting the same event updates properties on the existing
// edge rather than creating a parallel edge. This matches the openCypher MERGE
// semantics shared by Neo4j, Memgraph, Apache Age, and RedisGraph.
//
// Direction matters: an edge From A To B is distinct from an edge From B To A.
// Model bidirectional relationships as two edges or choose the canonical
// direction your queries traverse.
type EdgeRef struct {
	Type string
	From NodeRef
	To   NodeRef
}

// GraphSink is the transactional write context passed to graph projection
// handlers. All writes performed through a sink during a single
// [GraphProjection.Handle] call commit atomically — if the handler returns an
// error, every write is rolled back.
//
// Handlers never touch a driver directly. The driver (Neo4j, Memgraph,
// in-memory) is chosen at deployment time when the projection is constructed,
// not when the handler is written. This is what makes graph projections
// portable across backends: the same handler code merges nodes and edges on
// any driver that implements [GraphDriver].
//
// All methods are idempotent (MERGE semantics). Re-projecting the same event
// is a no-op, which makes catch-up/replay safe — the same guarantee the
// relational and document tiers provide.
type GraphSink interface {
	// MergeNode upserts the node identified by ref, setting props on it.
	// Existing properties not in props are preserved; properties in props
	// overwrite. The node is created if it does not exist.
	MergeNode(ref NodeRef, props map[string]any) error

	// MergeEdge upserts the directed edge identified by ref, setting props on
	// it. Both endpoint nodes are created if they do not exist (with no
	// properties beyond their key). This relaxes ordering constraints on
	// handlers — a handler may merge an edge before explicitly merging its
	// endpoints.
	MergeEdge(ref EdgeRef, props map[string]any) error

	// RemoveNode deletes the node and every edge incident to it.
	RemoveNode(ref NodeRef) error

	// RemoveEdge deletes the edge. Endpoint nodes are untouched.
	RemoveEdge(ref EdgeRef) error

	// SetNodeProperty sets a single property on an existing node. Use it for
	// sparse updates that should not clobber other properties (MergeNode with
	// a one-entry props map achieves the same effect; this method is the
	// explicit, self-documenting form).
	SetNodeProperty(ref NodeRef, prop string, value any) error
}

// GraphDriver is the backend interface that [GraphProjection] runs against.
// Implementations: [MemoryDriver] (in-memory, tests), and consumer-pulled
// drivers (Neo4j, Memgraph, Apache Age) in sibling modules.
//
// RunInTx executes fn inside a graph transaction. All GraphSink writes issued
// by fn commit atomically when fn returns nil and roll back when it returns an
// error. This is the graph analogue of BEGIN/COMMIT/ROLLBACK in SQL.
//
// The sink passed to fn is only valid for the duration of fn; using it after
// fn returns is undefined behaviour.
type GraphDriver interface {
	RunInTx(fn func(GraphSink) error) error
	Close() error
}

// Compile-time assertions that the reference implementation satisfies the
// interfaces. MemoryDriver is defined in memory.go.
var (
	_ GraphDriver = (*MemoryDriver)(nil)
)
