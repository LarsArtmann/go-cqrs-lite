package graph

import (
	"fmt"
	"maps"
	"sync"
)

// MemoryDriver is an in-memory [GraphDriver] — the reference implementation.
//
// It stores nodes and edges in an adjacency-list structure protected by a
// mutex. Transactions snapshot the graph on Begin, apply writes to the
// snapshot, and atomically swap it back on commit; a returning error discards
// the snapshot, leaving the published graph untouched. This gives real
// atomicity (not just the illusion of it) — the same guarantee a real graph
// database's transaction provides.
//
// MemoryDriver is safe for concurrent use. It is suitable for tests and for
// single-process local-first applications that want graph-shaped reads without
// deploying a graph database.
type MemoryDriver struct {
	mu     sync.Mutex
	data   *graphData
	schema *Schema
}

// MemoryDriverOption configures a [MemoryDriver].
type MemoryDriverOption func(*MemoryDriver)

// WithDriverSchema attaches a [Schema] to the driver for standalone use (without
// a [GraphProjection]). Every write through [MemoryDriver.RunInTx] is validated
// against the schema before it touches the graph. For the common case of a
// projection with schema validation, use [WithSchema] on [NewGraphProjection]
// instead — it validates at the projection boundary regardless of driver.
func WithDriverSchema(schema *Schema) MemoryDriverOption {
	return func(d *MemoryDriver) {
		d.schema = schema
	}
}

// NewMemoryDriver constructs an empty in-memory graph.
// Options allow attaching a [Schema] for write validation.
func NewMemoryDriver(opts ...MemoryDriverOption) *MemoryDriver {
	d := &MemoryDriver{
		mu:   sync.Mutex{},
		data: newGraphData(),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// RunInTx executes fn against a transactional snapshot. All writes commit
// atomically when fn returns nil; any error discards the snapshot.
func (d *MemoryDriver) RunInTx(fn func(GraphSink) error) error {
	d.mu.Lock()
	snapshot := d.data.clone()
	d.mu.Unlock()

	sink := wrapWithSchema(&memorySink{data: snapshot}, d.schema)

	if err := fn(sink); err != nil {
		return err // snapshot discarded
	}

	d.mu.Lock()
	d.data = snapshot
	d.mu.Unlock()

	return nil
}

// Close is a no-op for the in-memory driver (no resources to release).
func (*MemoryDriver) Close() error { return nil }

// Snapshot exposes a read-only copy of the graph for queries and assertions.
// Tests use it to verify projection outcomes without going through the sink.
// The returned snapshot is a deep copy; mutating it does not affect the driver.
func (d *MemoryDriver) Snapshot() *graphData {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.data.clone()
}

// nodeKey is the identity hash for a node within the in-memory store.
type nodeKey struct {
	label   string
	keyProp string
	keyVal  string
}

func (r NodeRef) key() (nodeKey, error) {
	if err := r.validate(); err != nil {
		return nodeKey{}, err
	}

	return nodeKey{label: r.Label, keyProp: r.KeyProp, keyVal: fmt.Sprint(r.KeyValue)}, nil
}

func (r NodeRef) validate() error {
	if r.Label == "" {
		return errEmptyLabel
	}

	if r.KeyProp == "" {
		return errEmptyKeyProp
	}

	return nil
}

// edgeKey is the identity hash for a directed edge.
type edgeKey struct {
	typ  string
	from nodeKey
	to   nodeKey
}

func (r EdgeRef) key() (edgeKey, error) {
	if r.Type == "" {
		return edgeKey{}, errEmptyEdgeType
	}

	from, err := r.From.key()
	if err != nil {
		return edgeKey{}, fmt.Errorf("edge from: %w", err)
	}

	to, err := r.To.key()
	if err != nil {
		return edgeKey{}, fmt.Errorf("edge to: %w", err)
	}

	return edgeKey{typ: r.Type, from: from, to: to}, nil
}

// node is a stored node: its identity key plus a property map.
type node struct {
	key   nodeKey
	props map[string]any
}

// edge is a stored directed edge.
type edge struct {
	key   edgeKey
	props map[string]any
}

// graphData is the mutable graph state. All methods are NOT goroutine-safe;
// callers (MemoryDriver or a transaction snapshot) must hold the owning mutex.
type graphData struct {
	nodes map[nodeKey]*node
	edges map[edgeKey]*edge
}

func newGraphData() *graphData {
	return &graphData{
		nodes: make(map[nodeKey]*node),
		edges: make(map[edgeKey]*edge),
	}
}

// clone produces a deep copy: new maps, new node/edge structs, new property
// maps. Mutating the clone does not affect the original.
func (g *graphData) clone() *graphData {
	c := &graphData{
		nodes: make(map[nodeKey]*node, len(g.nodes)),
		edges: make(map[edgeKey]*edge, len(g.edges)),
	}

	for k, n := range g.nodes {
		c.nodes[k] = &node{key: n.key, props: copyProps(n.props)}
	}

	for k, e := range g.edges {
		c.edges[k] = &edge{key: e.key, props: copyProps(e.props)}
	}

	return c
}

func copyProps(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	return maps.Clone(src)
}

// memorySink is the GraphSink handed to a handler inside a MemoryDriver
// transaction. It writes to a private snapshot that commits on RunInTx success.
type memorySink struct {
	data *graphData
}

func (s *memorySink) MergeNode(ref NodeRef, props map[string]any) error {
	k, err := ref.key()
	if err != nil {
		return err
	}

	n, ok := s.data.nodes[k]
	if !ok {
		n = &node{key: k, props: make(map[string]any)}
		s.data.nodes[k] = n
	}

	maps.Copy(n.props, props)

	return nil
}

func (s *memorySink) MergeEdge(ref EdgeRef, props map[string]any) error {
	k, err := ref.key()
	if err != nil {
		return err
	}

	// Auto-create endpoint nodes if absent (graph MERGE semantics).
	for _, endpoint := range []NodeRef{ref.From, ref.To} {
		endpKey, err := endpoint.key()
		if err != nil {
			return err
		}

		if _, ok := s.data.nodes[endpKey]; !ok {
			s.data.nodes[endpKey] = &node{key: endpKey, props: make(map[string]any)}
		}
	}

	e, ok := s.data.edges[k]
	if !ok {
		e = &edge{key: k, props: make(map[string]any)}
		s.data.edges[k] = e
	}

	maps.Copy(e.props, props)

	return nil
}

func (s *memorySink) RemoveNode(ref NodeRef) error {
	k, err := ref.key()
	if err != nil {
		return err
	}

	delete(s.data.nodes, k)

	// Remove every incident edge (either endpoint).
	for ek := range s.data.edges {
		if ek.from == k || ek.to == k {
			delete(s.data.edges, ek)
		}
	}

	return nil
}

func (s *memorySink) RemoveEdge(ref EdgeRef) error {
	k, err := ref.key()
	if err != nil {
		return err
	}

	delete(s.data.edges, k)

	return nil
}

func (s *memorySink) SetNodeProperty(ref NodeRef, prop string, value any) error {
	k, err := ref.key()
	if err != nil {
		return err
	}

	n, ok := s.data.nodes[k]
	if !ok {
		// MERGE-then-SET semantics: create the node if absent.
		n = &node{key: k, props: make(map[string]any)}
		s.data.nodes[k] = n
	}

	n.props[prop] = value

	return nil
}
