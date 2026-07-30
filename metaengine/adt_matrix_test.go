package metaengine_test

import (
	"context"
	"encoding/json/v2"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// adtMatrixTest.go provides a unified, parameterized test harness that exercises
// all 7 ADTs (Map, Set, Counter, Graph, SortedMap, Log, Multimap) across every
// available engine (memory, SQLite). Each ADT scenario populates data via the
// engine's backend interface, reads it back, and asserts cross-engine parity.
//
// To add a new engine (e.g. Pebble), add it to engineFactories — all scenarios
// run automatically.

// engineFactory creates a fresh, isolated engine for testing.
type engineFactory struct {
	name    string
	create  func(t *testing.T) metaengine.Engine
	supports func(metaengine.Engine) bool // nil = supports all ADTs
}

// adtScenario describes a single ADT's setup + read + parity check.
type adtScenario struct {
	name     string
	requires string // backend interface name ("MapBackend", "SetBackend", etc.)
	setup    func(ctx context.Context, eng metaengine.Engine) error
	read     func(ctx context.Context, eng metaengine.Engine) (any, error)
	// canonicalize normalizes the result for cross-engine comparison
	// (e.g. sorting map keys, normalizing float representation).
	canonicalize func(any) string
}

// TestADTMatrix runs all 7 ADT scenarios across all engines and asserts
// cross-engine parity. This replaces the ad-hoc per-ADT parity tests.
func TestADTMatrix(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	factories := engineFactories()
	scenarios := adtScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			results := make(map[string]string, len(factories))

			for _, factory := range factories {
				factory := factory

				t.Run(factory.name, func(t *testing.T) {
					t.Parallel()

					eng := factory.create(t)
					defer eng.Close()

					if err := scenario.setup(ctx, eng); err != nil {
						t.Fatalf("%s/%s setup: %v", factory.name, scenario.name, err)
					}

					raw, err := scenario.read(ctx, eng)
					if err != nil {
						t.Fatalf("%s/%s read: %v", factory.name, scenario.name, err)
					}

					results[factory.name] = scenario.canonicalize(raw)
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
							scenario.name, engines[0], first, name, results[name])
					}
				}
			})
		})
	}
}

// engineFactories returns the set of engines to test. Pebble is in a separate
// module and can be added here when its tests import this test helper.
func engineFactories() []engineFactory {
	return []engineFactory{
		{
			name:   "memory",
			create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			name:   "sqlite",
			create: func(t *testing.T) metaengine.Engine { return newIsolatedSQLiteEngine(t) },
		},
	}
}

// adtScenarios returns the 7-ADT test matrix. Each scenario exercises one ADT
// via its backend interface (not the Store/Execute path — that's covered by the
// cross_engine_meta_test.go Ginkgo suite).
func adtScenarios() []adtScenario {
	ctx := context.Background()
	_ = ctx

	return []adtScenario{
		// --- Map ADT: MapSet + MapGet ---
		{
			name:     "Map",
			requires: "MapBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
				mb := eng.(metaengine.MapBackend)
				if err := mb.MapSet(ctx, "users", "u1", map[string]any{"name": "Alice", "age": float64(30)}); err != nil {
					return err
				}

				return mb.MapSet(ctx, "users", "u2", map[string]any{"name": "Bob", "age": float64(25)})
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				mb := eng.(metaengine.MapBackend)
				v, _, err := mb.MapGet(ctx, "users", "u1")

				return v, err
			},
			canonicalize: canonicalizeAny,
		},

		// --- Set ADT: SetAdd + SetContains ---
		{
			name:     "Set",
			requires: "SetBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
				sb := eng.(metaengine.SetBackend)
				for _, k := range []string{"apple", "banana", "cherry"} {
					if err := sb.SetAdd(ctx, "fruits", k); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sb := eng.(metaengine.SetBackend)
				results := make(map[string]bool)
				for _, k := range []string{"apple", "banana", "cherry", "date"} {
					contains, err := sb.SetContains(ctx, "fruits", k)
					if err != nil {
						return nil, err
					}

					results[k] = contains
				}

				return results, nil
			},
			canonicalize: canonicalizeAny,
		},

		// --- Counter ADT: CounterIncrement + CounterGet ---
		{
			name:     "Counter",
			requires: "CounterBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
				cb := eng.(metaengine.CounterBackend)
				deltas := []metaengine.Delta{
					{"alpha": 1, "beta": 5},
					{"alpha": 2, "gamma": 3},
					{"beta": -3, "gamma": 1},
					{"alpha": 10},
				}
				for _, d := range deltas {
					if err := cb.CounterIncrement(ctx, "counters", d); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				cb := eng.(metaengine.CounterBackend)
				return cb.CounterGet(ctx, "counters")
			},
			canonicalize: canonicalizeCounter,
		},

		// --- Graph ADT: GraphAddEdge + GraphNeighbors ---
		{
			name:     "Graph",
			requires: "GraphBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
				gb := eng.(metaengine.GraphBackend)
				edges := []metaengine.Edge{
					{From: "A", To: "B"},
					{From: "A", To: "C"},
					{From: "B", To: "D"},
				}
				for _, e := range edges {
					if err := gb.GraphAddEdge(ctx, "graph", e); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				gb := eng.(metaengine.GraphBackend)
				return gb.GraphNeighbors(ctx, "graph", "A", 1)
			},
			canonicalize: canonicalizeNeighbors,
		},

		// --- SortedMap ADT: MapSet + MapScan (filter + sort) ---
		{
			name:     "SortedMap",
			requires: "ScanBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
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
					if err := mb.MapSet(ctx, "sorted", item.key, item.value); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				sb := eng.(metaengine.ScanBackend)
				return sb.MapScan(ctx, "sorted",
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
			canonicalize: canonicalizeScanResults,
		},

		// --- Log ADT: LogAppend + LogTail ---
		{
			name:     "Log",
			requires: "LogBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
				lb := eng.(metaengine.LogBackend)
				for _, v := range []string{"e1", "e2", "e3", "e4", "e5"} {
					if err := lb.LogAppend(ctx, "log", v); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				lb := eng.(metaengine.LogBackend)
				return lb.LogTail(ctx, "log", 3)
			},
			canonicalize: canonicalizeAny,
		},

		// --- Multimap ADT: MultiAdd + MultiGet ---
		{
			name:     "Multimap",
			requires: "MultimapBackend",
			setup: func(ctx context.Context, eng metaengine.Engine) error {
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
					if err := mb.MultiAdd(ctx, "tasks_by_user", e.key, e.value); err != nil {
						return err
					}
				}

				return nil
			},
			read: func(ctx context.Context, eng metaengine.Engine) (any, error) {
				mb := eng.(metaengine.MultimapBackend)
				return mb.MultiGet(ctx, "tasks_by_user", "alice")
			},
			canonicalize: canonicalizeAny,
		},
	}
}

// --- Canonicalization helpers for cross-engine parity ---

func canonicalizeAny(v any) string {
	if v == nil {
		return "<nil>"
	}

	return canonicalizeValue(v)
}

// canonicalizeValue produces a deterministic string representation of any value,
// sorting map keys so cross-engine comparison is order-independent. This is
// necessary because encoding/json/v2 does not sort map keys alphabetically.
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

func canonicalizeCounter(v any) string {
	counts, ok := v.(map[string]int64)
	if !ok {
		return canonicalizeAny(v)
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

func canonicalizeNeighbors(v any) string {
	neighbors, ok := v.([]any)
	if !ok {
		return canonicalizeAny(v)
	}

	strs := make([]string, 0, len(neighbors))
	for _, n := range neighbors {
		strs = append(strs, canonicalizeAny(n))
	}

	sort.Strings(strs)

	return strings.Join(strs, ",")
}

func canonicalizeScanResults(v any) string {
	results, ok := v.([]any)
	if !ok {
		return canonicalizeAny(v)
	}

	var b strings.Builder
	for _, r := range results {
		b.WriteString(canonicalizeAny(r))
		b.WriteString(";")
	}

	return b.String()
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
