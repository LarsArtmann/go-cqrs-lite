// Package enginetest provides shared test scenarios for metaengine Engine
// implementations. Each scenario is a self-contained test that exercises a
// common backend feature (MapScan, Watcher/Replay, etc.) and can be invoked
// against any engine that implements the required backend interface.
//
// Engines that share the same backend semantics (DuckDB, Postgres, SQLite ...
// for SQL backends; Badger, Pebble, RocksDB ... for LSM backends) typically
// need to verify the same set of behavioural contracts. The helpers in this
// package capture those contracts so each engine's tests can call them with
// a one-liner instead of duplicating the test body.
package enginetest

import (
	"context"
	"reflect"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ScanBackendItem is the canonical test product/fixture used by ScanBackend
// filters, sorts, and keyset pagination tests. It is exported so callers can
// reference the same field names in their own assertions.
type ScanBackendItem struct {
	Name     string
	Category string
	Price    float64
}

// ScanBackendProducts returns the canonical 5-item fixture for ScanBackend
// tests. The fixture is intentionally non-trivial — it covers multiple
// categories (fruit, veg, snack) and a non-monotonic price ordering so that
// sort + filter + keyset pagination can be meaningfully distinguished.
func ScanBackendProducts() []ScanBackendItem {
	return []ScanBackendItem{
		{Name: "apple", Category: "fruit", Price: 1.50},
		{Name: "banana", Category: "fruit", Price: 0.75},
		{Name: "carrot", Category: "veg", Price: 0.99},
		{Name: "donut", Category: "snack", Price: 2.00},
		{Name: "eggplant", Category: "veg", Price: 1.25},
	}
}

// RunScanBackendTest exercises the standard ScanBackend.MapScan contract:
//  1. unfiltered scan returns all 5 items
//  2. filter by category=fruit returns 2 items
//  3. sort by price descending, limit 3 → first item is "donut" (price 2.00)
//  4. keyset pagination cursor = donut (price 2.00), limit 2 → returns 2 items
//
// The engine must already implement both MapBackend and ScanBackend. The
// caller is responsible for closing the engine and choosing the fixture
// collection name (typically "products").
func RunScanBackendTest(t *testing.T, eng metaengine.Engine, collection string) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("engine %T does not implement MapBackend", eng)
	}

	sb, ok := eng.(metaengine.ScanBackend)
	if !ok {
		t.Fatalf("engine %T does not implement ScanBackend", eng)
	}

	ctx := context.Background()
	items := ScanBackendProducts()

	for _, item := range items {
		if err := mb.MapSet(ctx, collection, item.Name, item); err != nil {
			t.Fatal(err)
		}
	}

	// Test 1: No filter, no sort — should return all 5.
	results, err := sb.MapScan(ctx, collection, nil, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != len(items) {
		t.Fatalf("no filter: expected %d results, got %d", len(items), len(results.Items))
	}

	// Test 2: Filter by category=fruit — should return 2.
	fruitFilter := func(item any) bool {
		m, ok := item.(map[string]any)
		if !ok {
			return false
		}

		return m["Category"] == "fruit"
	}

	results, err = sb.MapScan(ctx, collection, fruitFilter, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("filter fruit: expected 2 results, got %d", len(results.Items))
	}

	// Test 3: Sort by price descending, limit 3.
	priceDesc := func(a, b any) int {
		am, _ := a.(map[string]any)
		bm, _ := b.(map[string]any)
		pa, _ := am["Price"].(float64)
		pb, _ := bm["Price"].(float64)

		switch {
		case pa < pb:
			return 1
		case pa > pb:
			return -1
		default:
			return 0
		}
	}

	results, err = sb.MapScan(ctx, collection, nil, priceDesc, nil, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 3 {
		t.Fatalf("limit 3: expected 3, got %d", len(results.Items))
	}

	first, _ := results.Items[0].(map[string]any)
	if first["Name"] != "donut" {
		t.Errorf("sorted desc: first item = %v, want donut", first["Name"])
	}

	// Test 4: Keyset pagination — cursor = donut (price 2.00), limit 2.
	results, err = sb.MapScan(ctx, collection, nil, priceDesc, map[string]any{"Price": 2.0}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("pagination: expected 2, got %d", len(results.Items))
	}
}

// WatcherReplaySetup wires the planner for a single-collection, single-event
// Watcher + replay test. The factory boils the engine-specific bits down to
// four inputs: the planner query, the store, the watcher factory, and the
// apply callback.
type WatcherReplaySetup[V any] struct {
	// Collection is the planner's collection name.
	Collection string
	// Build returns the planned store and a watcher over the planner's
	// collection. The store is held for the lifetime of the test.
	Build func(t *testing.T, eng metaengine.Engine) (*metaengine.Store, *metaengine.Watcher[V])
	// Apply is the engine-specific apply call (e.g. store.Apply(ctx, type, payload)).
	Apply func(ctx context.Context, store *metaengine.Store, payload V) error
}

// RunWatcherReplayTest exercises the standard Watcher + Replay contract:
//
//  1. The watcher's first sequenced notification carries the apply's value
//     and a non-zero seq (regression guard for pre-fix replayShim bug).
//  2. The ring buffer catches up: replay(0) returns the same single entry.
//
// expectedID is the ID asserted on the live and replayed values. seqTimeout
// is the duration to wait for the watcher's first notification (Postgres may
// need a longer timeout than DuckDB).
func RunWatcherReplayTest[V any](
	t *testing.T,
	eng metaengine.Engine,
	setup WatcherReplaySetup[V],
	payload V,
	expectedID string,
	seqTimeout time.Duration,
) {
	t.Helper()

	store, watcher := setup.Build(t, eng)
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	replay := watcher.WithReplay(100)
	seqCh := watcher.WatchWithSeq(ctx, nil)

	if err := setup.Apply(ctx, store, payload); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	select {
	case sv := <-seqCh:
		if !idsEqual(sv.Value, expectedID) {
			t.Errorf("expected %q, got %v", expectedID, idOf(sv.Value))
		}
		if sv.Seq == 0 {
			t.Fatal("expected non-zero seq — replayShim.recordValue silently failed (pre-fix bug)")
		}
	case <-time.After(seqTimeout):
		t.Fatalf("timeout waiting for %s watcher seq notification", engineName(eng))
	}

	entries := replay.Replay(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 replay entry, got %d", len(entries))
	}

	if !idsEqual(entries[0].Value, expectedID) {
		t.Errorf("replay entry: expected %q, got %v", expectedID, idOf(entries[0].Value))
	}
}

// idsEqual reports whether the value carries the expectedID via its "ID" field
// (any kind of string-shaped value like watcherTaskID, plain string, etc.).
func idsEqual(v any, expectedID string) bool {
	return idOf(v) == expectedID
}

// idOf extracts an "ID" field from v via reflection. Handles both
// map[string]any (DuckDB JSON-decoded values) and struct values with an
// exported ID field.
func idOf(v any) string {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Map {
		id := rv.MapIndex(reflect.ValueOf("ID"))
		if id.IsValid() && id.Kind() == reflect.String {
			return id.String()
		}

		return ""
	}

	if rv.Kind() == reflect.Struct {
		f := rv.FieldByName("ID")
		if !f.IsValid() {
			return ""
		}

		switch f.Kind() {
		case reflect.String:
			return f.String()
		default:
			if f.Type().Implements(stringerType) {
				if s, ok := f.Interface().(interface{ String() string }); ok {
					return s.String()
				}
			}
		}
	}

	return ""
}

var stringerType = reflect.TypeOf((*interface{ String() string })(nil)).Elem()

// engineName returns a short stable name for the engine so test failures
// carry enough context without reflecting on the concrete pointer type.
func engineName(eng metaengine.Engine) string {
	if eng == nil {
		return "nil"
	}

	t := reflect.TypeOf(eng)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name()
}
