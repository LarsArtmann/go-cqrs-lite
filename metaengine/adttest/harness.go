// Package adttest provides a parameterized test harness that exercises all 10
// ADTs (Map, Set, Counter, Graph, SortedMap, Log, Multimap, Vector, Search,
// Spatial) across every available engine implementation, verifying behavioral
// parity.
//
// Usage from any engine module's tests:
//
//	func TestADTMatrix(t *testing.T) {
//	    adttest.RunMatrix(t, []adttest.Factory{
//	        {Name: "memory", Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() }},
//	        {Name: "sqlite", Create: func(t *testing.T) metaengine.Engine { return newIsolatedSQLiteEngine(t) }},
//	        {Name: "pebble", Create: func(t *testing.T) metaengine.Engine {
//	            eng, _ := pebbleengine.NewPebbleEngine("")
//	            return eng
//	        }},
//	    })
//	}
package adttest

import (
	"context"
	"encoding/json/v2"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphBackend is the local graph dispatch contract for test scenarios.
// Engines with graph support implement this structurally: the memory engine
// natively (universal fallback), dgraphengine natively, and irohengine by
// forwarding to its wrapped local engine. Consumers add graph support via
// graphadapter (ADR-0113).
type graphBackend interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// graphRemovalBackend extends graphBackend with tombstone-driven edge
// deletion (ADR-0114 style) for the GraphRemove scenario.
type graphRemovalBackend interface {
	graphBackend
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
}

// undirectedGraphBackend extends graphBackend with both-direction traversal
// for the GraphUndirected scenario.
type undirectedGraphBackend interface {
	graphBackend
	GraphNeighborsUndirected(
		ctx context.Context,
		collection string,
		node any,
		depth int,
	) ([]any, error)
}

// Factory creates a fresh, isolated engine for testing.
type Factory struct {
	Name     string
	Create   func(t *testing.T) metaengine.Engine
	Supports func(metaengine.Engine) bool // nil = supports all ADTs

	// PreClean optionally removes persistent state for a collection name
	// before the matrix starts (drop the planned table, delete meta_map
	// rows). Persistent databases (userspace MariaDB cqrs_test) need this
	// for fixed collection names; fresh-container/memory engines pass nil.
	PreClean func(t *testing.T, collection string)
}

// Scenario describes a single ADT's setup + read + parity check.
type Scenario struct {
	Name     string
	Requires string // backend interface name ("MapBackend", "SetBackend", etc.)
	Setup    func(ctx context.Context, eng metaengine.Engine) error
	Read     func(ctx context.Context, eng metaengine.Engine) (any, error)
	// Canonicalize normalizes the result for cross-engine comparison
	// (e.g. sorting map keys, normalizing float representation).
	Canonicalize func(any) string
}

// backendInterfaces maps Scenario.Requires names to their reflect.Type for
// automatic capability skipping. When an engine does not implement the
// required backend interface, RunMatrix skips that subtest instead of
// panicking on a failed type assertion inside Setup/Read.
var backendInterfaces = map[string]reflect.Type{ //nolint:gochecknoglobals // immutable lookup table
	"MapBackend":          reflect.TypeFor[metaengine.MapBackend](),
	"SetBackend":          reflect.TypeFor[metaengine.SetBackend](),
	"CounterBackend":      reflect.TypeFor[metaengine.CounterBackend](),
	"GraphBackend":        reflect.TypeFor[graphBackend](),
	"GraphRemovalBackend": reflect.TypeFor[graphRemovalBackend](),
	"UndirectedGraph":     reflect.TypeFor[undirectedGraphBackend](),
	"ScanBackend":         reflect.TypeFor[metaengine.ScanBackend](),
	"LogBackend":          reflect.TypeFor[metaengine.LogBackend](),
	"MultimapBackend":     reflect.TypeFor[metaengine.MultimapBackend](),
	"StreamLogBackend":    reflect.TypeFor[metaengine.StreamLogBackend](),
	"VectorBackend":       reflect.TypeFor[metaengine.VectorBackend](),
	"VectorFilterBackend": reflect.TypeFor[metaengine.VectorFilterBackend](),
	"SearchBackend":       reflect.TypeFor[metaengine.SearchBackend](),
	"SpatialBackend":      reflect.TypeFor[metaengine.SpatialBackend](),
}

// RunMatrix runs all ADT scenarios across all engine factories and asserts
// cross-engine parity. Each scenario runs in a subtest; each engine runs
// in a nested subtest. Engines that do not implement a scenario's required
// backend interface (Scenario.Requires) are skipped automatically. If
// Factory.Supports is non-nil, it is called as an additional capability gate.
// Cross-engine parity is checked in a cleanup hook.
//
// Scenario collections are scoped per Scenarios() invocation (each fixed
// name gains a "_<r><unix-nano>" suffix) and scenario assertions are
// absolute (exact values, versions, counts). RunMatrix calls Scenarios()
// once per run, so every factory in that run shares one suffix — required
// by the cross-engine parity check — while separate runs (reruns,
// -count>1 repeats inside one test binary) land in disjoint collections,
// so engines backed by a server shared across runs never observe
// accumulated state. Factories sharing ONE server WITHIN a run (e.g. two
// factories over the same database) remain unsupported: additive scenarios
// (Counter, Log, StreamLog) would double-apply and break parity.
func RunMatrix(t *testing.T, factories []Factory) {
	t.Helper()

	ctx := context.Background()
	scenarios := Scenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			results := make(map[string]string, len(factories))

			var mu sync.Mutex

			for _, factory := range factories {
				t.Run(factory.Name, func(t *testing.T) {
					t.Parallel()

					eng := factory.Create(t)
					defer metaengine.DeferClose(eng)

					// Auto-skip if the engine doesn't implement the required backend.
					if iface, ok := backendInterfaces[scenario.Requires]; ok {
						if !reflect.TypeOf(eng).Implements(iface) {
							t.Skipf("%s does not implement %s", factory.Name, scenario.Requires)

							return
						}
					}

					// Additional custom capability gate.
					if factory.Supports != nil && !factory.Supports(eng) {
						t.Skipf("%s skipped by Factory.Supports", factory.Name)

						return
					}

					if err := scenario.Setup(ctx, eng); err != nil {
						t.Fatalf("%s/%s setup: %v", factory.Name, scenario.Name, err)
					}

					raw, err := scenario.Read(ctx, eng)
					if err != nil {
						t.Fatalf("%s/%s read: %v", factory.Name, scenario.Name, err)
					}

					mu.Lock()
					results[factory.Name] = scenario.Canonicalize(raw)
					mu.Unlock()
				})
			}

			// Cross-engine parity: all engines must produce the same canonical result.
			t.Cleanup(func() {
				if len(results) < 2 {
					return
				}

				var engines []string
				for name := range results {
					engines = append(engines, name)
				}

				sort.Strings(engines)
				first := results[engines[0]]

				for _, name := range engines[1:] {
					if results[name] != first {
						t.Errorf("%s: cross-engine divergence\n  %s=%s\n  %s=%s",
							scenario.Name, engines[0], first, name, results[name])
					}
				}
			})
		})
	}
}

// Scenarios returns the 11-ADT test matrix. Each scenario exercises one ADT
// via its backend interface (not the Store/Execute path — that's covered by
// the cross_engine_meta_test.go Ginkgo suite).
func Scenarios() []Scenario { //nolint:maintidx // 11-ADT test matrix
	runSuffix := "r" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// col scopes a fixed scenario collection name to this Scenarios()
	// invocation, isolating repeated runs against shared servers (see the
	// RunMatrix doc comment for the parity-vs-isolation contract).
	col := func(name string) string { return name + "_" + runSuffix }

	return []Scenario{
		// --- Map ADT: MapSet + MapGet ---
		{
			Name:     "Map",
			Requires: "MapBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				mb := eng.(metaengine.MapBackend)
				if err := mb.MapSet(
					ctx,
					col("users"),
					"u1",
					map[string]any{"name": "Alice", "age": float64(30)},
				); err != nil {
					return err //nolint:wrapcheck
				}

				return mb.MapSet(
					ctx,
					col("users"),
					"u2",
					map[string]any{"name": "Bob", "age": float64(25)},
				)
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				mb := eng.(metaengine.MapBackend)
				v, _, err := mb.MapGet(ctx, col("users"), "u1")

				return v, err //nolint:wrapcheck
			},
			Canonicalize: CanonicalizeAny,
		},

		// --- Set ADT: SetAdd + SetContains ---
		{
			Name:     "Set",
			Requires: "SetBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				sb := eng.(metaengine.SetBackend)
				for _, k := range []string{"apple", "banana", "cherry"} {
					if err := sb.SetAdd(ctx, col("fruits"), k); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sb := eng.(metaengine.SetBackend)
				results := make(map[string]bool)
				for _, k := range []string{"apple", "banana", "cherry", "date"} {
					contains, err := sb.SetContains(ctx, col("fruits"), k)
					if err != nil {
						return nil, err //nolint:wrapcheck
					}

					results[k] = contains
				}

				return results, nil
			},
			Canonicalize: CanonicalizeAny,
		},

		// --- Counter ADT: CounterIncrement + CounterGet ---
		{
			Name:     "Counter",
			Requires: "CounterBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				cb := eng.(metaengine.CounterBackend)
				deltas := []metaengine.Delta{
					{"alpha": 1, "beta": 5},
					{"alpha": 2, "gamma": 3},
					{"beta": -3, "gamma": 1},
					{"alpha": 10},
				}
				for _, d := range deltas {
					if err := cb.CounterIncrement(ctx, col("counters"), d); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				cb := eng.(metaengine.CounterBackend)

				return cb.CounterGet(ctx, col("counters"))
			},
			Canonicalize: CanonicalizeCounter,
		},

		// --- Graph ADT: GraphAddEdge + GraphNeighbors ---
		{
			Name:     "Graph",
			Requires: "GraphBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				gb := eng.(graphBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "A", To: "C"},
					{From: "B", To: "D"},
				}
				for _, e := range edges {
					if err := gb.GraphAddEdge(ctx, col("graph"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gb := eng.(graphBackend)

				return gb.GraphNeighbors(ctx, col("graph"), "A", 1)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- Graph ADT: tombstone-driven edge removal (ADR-0114 style) ---
		{
			Name:     "GraphRemove",
			Requires: "GraphRemovalBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				gr := eng.(graphRemovalBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "A", To: "C"},
				}
				for _, e := range edges {
					if err := gr.GraphAddEdge(ctx, col("graph_rm"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				// Tombstone: retract A→B, then idempotently re-remove it.
				if err := gr.GraphRemoveEdge(
					ctx,
					col("graph_rm"),
					metaengine.Edge{From: "A", To: "B"},
				); err != nil {
					return err //nolint:wrapcheck
				}

				return gr.GraphRemoveEdge(
					ctx,
					col("graph_rm"),
					metaengine.Edge{From: "A", To: "B"},
				) //nolint:wrapcheck
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gr := eng.(graphRemovalBackend)

				return gr.GraphNeighbors(ctx, col("graph_rm"), "A", 1)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- Graph ADT: undirected traversal (both edge directions) ---
		{
			Name:     "GraphUndirected",
			Requires: "UndirectedGraph",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				ug := eng.(undirectedGraphBackend)
				edges := []metaengine.Edge{
					{From: "B", To: "A"}, // incoming edge for A
					{From: "A", To: "C"}, // outgoing edge for A
				}
				for _, e := range edges {
					if err := ug.GraphAddEdge(ctx, col("graph_und"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				ug := eng.(undirectedGraphBackend)

				return ug.GraphNeighborsUndirected(ctx, col("graph_und"), "A", 1)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- Graph ADT: depth-3 diamond (multi-path reachability canary) ---
		//
		// The recursive-CTE path (WITH RECURSIVE + UNION + DISTINCT) and the
		// iterative BFS fallback only diverge observably at depth > 2 when
		// nodes are reachable via multiple paths: here D is reachable at
		// depth 2 via BOTH A→B→D and A→C→D, and E requires a third hop.
		// Every engine must return exactly the deduplicated set {B,C,D,E} —
		// a duplicate D (missing DISTINCT in a CTE) or a missing E (frontier
		// lost in an iterative walk) fails cross-engine parity.
		{
			Name:     "GraphDepth3Diamond",
			Requires: "GraphBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				gb := eng.(graphBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "A", To: "C"},
					{From: "B", To: "D"},
					{From: "C", To: "D"},
					{From: "D", To: "E"},
				}
				for _, e := range edges {
					if err := gb.GraphAddEdge(ctx, col("graph_deep"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gb := eng.(graphBackend)

				return gb.GraphNeighbors(ctx, col("graph_deep"), "A", 3)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- Graph ADT: cycle termination + duplicate suppression ---
		//
		// A→B→C→A closes a cycle back to the start node; D→B re-enters the
		// cycle from a side branch. A depth-4 walk must terminate (no
		// infinite recursion in CTE mode, no frontier regrowth in iterative
		// mode), exclude the start node A, and return each reachable node
		// exactly once: {B,C,D}.
		{
			Name:     "GraphCycle",
			Requires: "GraphBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				gb := eng.(graphBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "A"},
					{From: "C", To: "D"},
					{From: "D", To: "B"},
				}
				for _, e := range edges {
					if err := gb.GraphAddEdge(ctx, col("graph_cycle"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gb := eng.(graphBackend)

				return gb.GraphNeighbors(ctx, col("graph_cycle"), "A", 4)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- Graph ADT: depth bound excludes deeper nodes ---
		//
		// A linear chain A→B→C→D→E with depth 2 must return only {B,C}:
		// D and E exist but sit beyond the bound. Catches engines whose
		// CTE depth predicate or iterative level counter is off by one.
		{
			Name:     "GraphDepthBound",
			Requires: "GraphBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				gb := eng.(graphBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "D"},
					{From: "D", To: "E"},
				}
				for _, e := range edges {
					if err := gb.GraphAddEdge(ctx, col("graph_bound"), e); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gb := eng.(graphBackend)

				return gb.GraphNeighbors(ctx, col("graph_bound"), "A", 2)
			},
			Canonicalize: CanonicalizeNeighbors,
		},

		// --- SortedMap ADT: MapSet + MapScan (filter + sort) ---
		{
			Name:     "SortedMap",
			Requires: "ScanBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				mb := eng.(metaengine.MapBackend)
				items := []struct {
					key   string
					value map[string]any
				}{
					{"s1", map[string]any{"status": "open", "priority": float64(3)}},
					{"s2", map[string]any{"status": "open", "priority": float64(1)}},
					{"s3", map[string]any{"status": "done", "priority": float64(2)}},
				}
				for _, item := range items {
					if err := mb.MapSet(ctx, col("sorted"), item.key, item.value); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sb := eng.(metaengine.ScanBackend)

				return sb.MapScan(
					ctx, col("sorted"),
					func(item any) bool {
						m, ok := item.(map[string]any)
						if !ok {
							return false
						}

						return m["status"] == "open"
					},
					func(a, b any) int {
						am, _ := a.(map[string]any)
						bm, _ := b.(map[string]any)
						ap, _ := am["priority"].(float64)
						bp, _ := bm["priority"].(float64)
						if ap < bp {
							return -1
						}
						if ap > bp {
							return 1
						}

						return 0
					},
					nil, 10,
				)
			},
			Canonicalize: CanonicalizeScanResults,
		},

		// --- Log ADT: LogAppend + LogTail ---
		{
			Name:     "Log",
			Requires: "LogBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				lb := eng.(metaengine.LogBackend)
				for _, v := range []string{"e1", "e2", "e3", "e4", "e5"} {
					if err := lb.LogAppend(ctx, col("log"), v); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				lb := eng.(metaengine.LogBackend)

				return lb.LogTail(ctx, col("log"), 3)
			},
			Canonicalize: CanonicalizeAny,
		},

		// --- Multimap ADT: MultiAdd + MultiGet ---
		{
			Name:     "Multimap",
			Requires: "MultimapBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				mb := eng.(metaengine.MultimapBackend)
				entries := []struct {
					key   string
					value string
				}{
					{"alice", "t1"},
					{"alice", "t2"},
					{"bob", "t3"},
					{"alice", "t4"},
				}
				for _, e := range entries {
					if err := mb.MultiAdd(ctx, col("tasks_by_user"), e.key, e.value); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				mb := eng.(metaengine.MultimapBackend)

				return mb.MultiGet(ctx, col("tasks_by_user"), "alice")
			},
			Canonicalize: CanonicalizeAny,
		},

		// --- StreamLog ADT: StreamAppend + StreamRead + JournalReadAll ---
		{
			Name:     "StreamLog",
			Requires: "StreamLogBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				slb := eng.(metaengine.StreamLogBackend)
				if err := slb.StreamAppend(
					ctx,
					col("events"),
					"s1",
					[]any{"e1", "e2", "e3"},
				); err != nil {
					return err //nolint:wrapcheck
				}

				return slb.StreamAppend(ctx, col("events"), "s2", []any{"e4", "e5"})
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				slb := eng.(metaengine.StreamLogBackend)
				values, err := slb.StreamRead(ctx, col("events"), "s1")
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				ver, err := slb.StreamVersion(ctx, col("events"), "s1")
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				return map[string]any{"values": values, "version": ver}, nil
			},
			Canonicalize: CanonicalizeAny,
		},

		// --- Vector ADT: VectorInsert + VectorSearch ---
		{
			Name:     "Vector",
			Requires: "VectorBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				vb := eng.(metaengine.VectorBackend)
				embeddings := []metaengine.Embedding{
					{ID: "v1", Values: []float32{1.0, 0.0, 0.0}},
					{ID: "v2", Values: []float32{0.9, 0.1, 0.0}},
					{ID: "v3", Values: []float32{0.0, 1.0, 0.0}},
				}
				for _, emb := range embeddings {
					if err := vb.VectorInsert(ctx, col("vectors"), emb); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				vb := eng.(metaengine.VectorBackend)

				return vb.VectorSearch(ctx, col("vectors"), []float32{1.0, 0.0, 0.0}, 2, "cosine")
			},
			Canonicalize: CanonicalizeVector,
		},

		// --- Vector ADT: metadata-filtered k-NN ---
		{
			Name:     "VectorFiltered",
			Requires: "VectorFilterBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				vf := eng.(metaengine.VectorFilterBackend)
				embeddings := []metaengine.Embedding{
					{
						ID:       "v1",
						Values:   []float32{1.0, 0.0, 0.0},
						Metadata: map[string]any{"tenant": "a"},
					},
					{
						ID:       "v2",
						Values:   []float32{0.9, 0.1, 0.0},
						Metadata: map[string]any{"tenant": "b"},
					},
					{
						ID:       "v3",
						Values:   []float32{0.0, 1.0, 0.0},
						Metadata: map[string]any{"tenant": "a"},
					},
				}
				for _, emb := range embeddings {
					if err := vf.VectorInsert(ctx, col("vectors_filtered"), emb); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				vf := eng.(metaengine.VectorFilterBackend)

				return vf.VectorSearchFiltered(
					ctx,
					col("vectors_filtered"),
					[]float32{1.0, 0.0, 0.0},
					2,
					"cosine",
					[]metaengine.VectorFilter{
						{Field: "tenant", Op: metaengine.FilterEq, Value: "a"},
					},
				)
			},
			Canonicalize: CanonicalizeVector,
		},

		// --- Search ADT: SearchInsert + SearchQuery ---
		{
			Name:     "Search",
			Requires: "SearchBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				sb := eng.(metaengine.SearchBackend)
				docs := []metaengine.IndexedText{
					{ID: "d1", Content: "the quick brown fox"},
					{ID: "d2", Content: "the lazy brown dog"},
					{ID: "d3", Content: "quick thinking saves time"},
				}
				for _, doc := range docs {
					if err := sb.SearchInsert(ctx, col("docs"), doc); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sb := eng.(metaengine.SearchBackend)

				return sb.SearchQuery(ctx, col("docs"), "quick", 5)
			},
			Canonicalize: CanonicalizeSearch,
		},

		// --- Spatial ADT: SpatialInsert + SpatialRange ---
		{
			Name:     "Spatial",
			Requires: "SpatialBackend",
			Setup: func(ctx context.Context, eng metaengine.Engine) error {
				sp := eng.(metaengine.SpatialBackend)
				points := []metaengine.Point{
					{ID: "p1", X: 13.4050, Y: 52.5200}, // Berlin
					{ID: "p2", X: 13.4100, Y: 52.5150}, // Near Berlin
					{ID: "p3", X: 2.3522, Y: 48.8566},  // Paris (far)
				}
				for _, pt := range points {
					if err := sp.SpatialInsert(ctx, col("places"), pt); err != nil {
						return err //nolint:wrapcheck
					}
				}

				return nil
			},
			Read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sp := eng.(metaengine.SpatialBackend)

				return sp.SpatialRange(ctx, col("places"), 13.4050, 52.5200, 10000, 5)
			},
			Canonicalize: CanonicalizeSpatial,
		},
	}
}

// --- Canonicalization helpers for cross-engine parity ---

// CanonicalizeAny produces a deterministic string representation of any value.
func CanonicalizeAny(v any) string {
	if v == nil {
		return "<nil>"
	}

	return canonicalizeValue(v)
}

// canonicalizeValue produces a deterministic string representation of any value,
// sorting map keys so cross-engine comparison is order-independent.
func canonicalizeValue(v any) string {
	switch val := v.(type) {
	case map[string]any:
		return canonicalizeStringMap(val)
	case map[string]bool:
		m := make(map[string]any, len(val))
		for k, b := range val {
			m[k] = b
		}

		return canonicalizeStringMap(m)
	case []any:
		var b strings.Builder
		b.WriteString("[")

		for i, item := range val {
			if i > 0 {
				b.WriteString(",")
			}

			b.WriteString(canonicalizeValue(item))
		}

		b.WriteString("]")

		return b.String()
	case string:
		return strconv.Quote(val)
	default:
		return mustJSON(v)
	}
}

// CanonicalizeCounter canonicalizes a counter map for cross-engine comparison.
func CanonicalizeCounter(v any) string {
	counts, ok := v.(map[string]int64)
	if !ok {
		return CanonicalizeAny(v)
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(mustJSON(counts[k]))
		b.WriteString(",")
	}

	return b.String()
}

// CanonicalizeNeighbors canonicalizes a neighbor list for cross-engine comparison.
func CanonicalizeNeighbors(v any) string {
	neighbors, ok := v.([]any)
	if !ok {
		return CanonicalizeAny(v)
	}

	strs := make([]string, 0, len(neighbors))
	for _, n := range neighbors {
		strs = append(strs, CanonicalizeAny(n))
	}

	sort.Strings(strs)

	return strings.Join(strs, ",")
}

// CanonicalizeScanResults canonicalizes scan results for cross-engine comparison.
func CanonicalizeScanResults(v any) string {
	result, ok := v.(metaengine.ScanResult)
	if !ok {
		return CanonicalizeAny(v)
	}

	var b strings.Builder
	for _, r := range result.Items {
		b.WriteString(CanonicalizeAny(r))
		b.WriteString(";")
	}

	return b.String()
}

// canonicalizeIDs projects result entries to their IDs, sorts them, and joins
// with commas. Distances/scores may vary between engines (float arithmetic,
// TF-IDF vs BM25) and ties may order differently, so comparisons use IDs only.
func canonicalizeIDs(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)

	return strings.Join(sorted, ",")
}

// CanonicalizeVector canonicalizes vector search results by sorted ID list.
// Distances may vary slightly between engines due to float arithmetic, and
// ties (equal distances) may be returned in different order by different
// engines, so we sort IDs for order-independent comparison.
func CanonicalizeVector(v any) string {
	results, ok := v.([]metaengine.VectorResult)
	if !ok {
		return CanonicalizeAny(v)
	}

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	return canonicalizeIDs(ids)
}

// CanonicalizeSearch canonicalizes full-text search results by ID list.
// Scores may vary between engines (TF-IDF vs BM25), so we compare IDs only.
func CanonicalizeSearch(v any) string {
	results, ok := v.([]metaengine.SearchResult)
	if !ok {
		return CanonicalizeAny(v)
	}

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	return canonicalizeIDs(ids)
}

// CanonicalizeSpatial canonicalizes spatial range results by ID list.
// Distances may vary slightly between engines, so we compare IDs (sorted,
// since different engines may return points at similar distances in different order).
func CanonicalizeSpatial(v any) string {
	results, ok := v.([]metaengine.SpatialResult)
	if !ok {
		return CanonicalizeAny(v)
	}

	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	return canonicalizeIDs(ids)
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<error>"
	}

	return string(b)
}

func canonicalizeStringMap(val map[string]any) string {
	keys := make([]string, 0, len(val))
	for k := range val {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{")

	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}

		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(canonicalizeValue(val[k]))
	}

	b.WriteString("}")

	return b.String()
}
