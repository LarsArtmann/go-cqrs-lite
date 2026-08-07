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
	"errors"
	"fmt"
	"reflect"
	"sync"
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

// RunPushdownTest exercises a single PushdownScan scenario. The setup
// (engine creation, skip on missing engine, seeding the fixture collection,
// type-asserting PushdownScan) is shared across all five scenarios in each
// engine's pushdown test file (filter, sort, filter+sort+limit, cursor, IN).
func RunPushdownTest(
	t *testing.T,
	eng metaengine.Engine,
	collection string,
	seedFixture func(t *testing.T, eng metaengine.Engine, col string),
	run func(t *testing.T, ctx context.Context, ps metaengine.PushdownScan),
) {
	t.Helper()

	seedFixture(t, eng, collection)

	ps, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Fatal("engine does not implement PushdownScan")
	}

	run(t, context.Background(), ps)
}

// RunStreamLogBackendTest exercises the standard StreamLogBackend contract:
//  1. StreamAppend to two streams — s1 (3 items) and s2 (1 item)
//  2. StreamRead returns the 3 items for s1
//  3. StreamVersion returns 3 for s1
//  4. JournalReadAll returns 4 total entries
//  5. JournalReadFrom(2, 0) returns 2 entries
//
// The engine must already implement StreamLogBackend. The caller is responsible
// for closing the engine (typically via t.Cleanup).
func RunStreamLogBackendTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	slb, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatalf("engine %T does not implement StreamLogBackend", eng)
	}

	ctx := context.Background()

	// Append to two streams.
	if err := slb.StreamAppend(ctx, "events", "s1", []any{"e1", "e2", "e3"}); err != nil {
		t.Fatalf("StreamAppend s1: %v", err)
	}

	if err := slb.StreamAppend(ctx, "events", "s2", []any{"e4"}); err != nil {
		t.Fatalf("StreamAppend s2: %v", err)
	}

	// Verify StreamRead.
	values, err := slb.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead s1: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	// Verify StreamVersion.
	ver, err := slb.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion s1: %v", err)
	}

	if ver != 3 {
		t.Fatalf("expected version 3, got %d", ver)
	}

	// Verify JournalReadAll.
	journal, err := slb.JournalReadAll(ctx, "events")
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}

	if len(journal) != 4 {
		t.Fatalf("expected 4 journal entries, got %d", len(journal))
	}

	// Verify JournalReadFrom.
	from2, err := slb.JournalReadFrom(ctx, "events", 2, 0)
	if err != nil {
		t.Fatalf("JournalReadFrom: %v", err)
	}

	if len(from2) != 2 {
		t.Fatalf("expected 2 entries after seq 2, got %d", len(from2))
	}
}

// RunAtomicAppenderTest exercises the standard AtomicAppender contract:
//  1. Append at version 0 → succeeds
//  2. Append at version 2 → succeeds
//  3. Append at version 0 (stale) → fails with ErrVersionConflict
//
// The engine must already implement AtomicAppender. The caller is responsible
// for closing the engine.
func RunAtomicAppenderTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	ap, ok := eng.(metaengine.AtomicAppender)
	if !ok {
		t.Fatalf("engine %T does not implement AtomicAppender", eng)
	}

	ctx := context.Background()

	// Append at version 0 → succeeds.
	if err := ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"a", "b"}); err != nil {
		t.Fatalf("StreamAppendExpected v0: %v", err)
	}

	// Append at version 2 → succeeds.
	if err := ap.StreamAppendExpected(ctx, "events", "s1", 2, []any{"c"}); err != nil {
		t.Fatalf("StreamAppendExpected v2: %v", err)
	}

	// Append at version 0 (stale) → fails with ErrVersionConflict.
	err := ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"d"})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
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
	for rv.Kind() == reflect.Pointer {
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
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Name()
}

// RunTransactionalTest exercises the standard Transactional (RunInTx) contract:
//  1. A successful transaction commits all writes
//  2. A failed transaction rolls back all writes
//  3. Writes inside a transaction are visible to reads within the same tx
//  4. CounterIncrement is atomic inside RunInTx (if CounterBackend is implemented)
//  5. StreamAppend is atomic inside RunInTx (if StreamLogBackend is implemented)
//
// The engine must implement Transactional and MapBackend. CounterBackend and
// StreamLogBackend are tested opportunistically when present. The caller is
// responsible for closing the engine.
//
//nolint:gocyclo // sequential assertions across 3 optional backends
func RunTransactionalTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	tx, ok := eng.(metaengine.Transactional)
	if !ok {
		t.Fatalf("engine %T does not implement Transactional", eng)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("engine %T does not implement MapBackend", eng)
	}

	ctx := context.Background()
	col := "tx_test_" + engineName(eng)

	// 1. Successful transaction commits.
	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		return mb.MapSet(ctx, col, "committed", "v1")
	})
	if err != nil {
		t.Fatalf("RunInTx commit path: %v", err)
	}

	val, found, err := mb.MapGet(ctx, col, "committed")
	if err != nil {
		t.Fatalf("MapGet after commit: %v", err)
	}

	if !found {
		t.Fatalf("expected key to exist after commit")
	}

	_ = val // value is JSON-decoded; just checking existence

	// 2. Failed transaction rolls back.
	sentinel := errors.New("rollback sentinel")

	err = tx.RunInTx(ctx, func(ctx context.Context) error {
		if e := mb.MapSet(ctx, col, "rolled-back", "v2"); e != nil {
			return e
		}

		// Read inside tx sees the write (for real-transaction engines).
		_, insideFound, insideErr := mb.MapGet(ctx, col, "rolled-back")
		if insideErr != nil {
			return insideErr
		}

		if !insideFound {
			t.Fatalf("expected write to be visible inside transaction")
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error from RunInTx, got %v", err)
	}

	// 3. Verify rollback: the key should NOT exist outside the tx.
	_, found, err = mb.MapGet(ctx, col, "rolled-back")
	if err != nil {
		t.Fatalf("MapGet after rollback: %v", err)
	}

	if found {
		t.Fatalf("expected key to NOT exist after rollback")
	}

	// 4. CounterIncrement inside RunInTx (if engine implements CounterBackend).
	if cb, ok := eng.(metaengine.CounterBackend); ok {
		runCounterTxSubtest(t, tx, cb, ctx, col+"_counter", sentinel)
	}

	// 5. StreamAppend inside RunInTx (if engine implements StreamLogBackend).
	if sb, ok := eng.(metaengine.StreamLogBackend); ok {
		runStreamTxSubtest(t, tx, sb, ctx, col+"_stream", sentinel)
	}

	// 6. MultiAdd inside RunInTx (if engine implements MultimapBackend).
	if mm, ok := eng.(metaengine.MultimapBackend); ok {
		runMultimapTxSubtest(t, tx, mm, ctx, col+"_multimap", sentinel)
	}

	// 7. LogAppend inside RunInTx (if engine implements LogBackend).
	if lb, ok := eng.(metaengine.LogBackend); ok {
		runLogTxSubtest(t, tx, lb, ctx, col+"_log", sentinel)
	}
}

func runCounterTxSubtest(
	t *testing.T, tx metaengine.Transactional, cb metaengine.CounterBackend,
	ctx context.Context, counterCol string, sentinel error,
) {
	t.Helper()

	if e := tx.RunInTx(ctx, func(ctx context.Context) error {
		return cb.CounterIncrement(ctx, counterCol, metaengine.Delta{"views": 5})
	}); e != nil {
		t.Fatalf("CounterIncrement in tx (commit): %v", e)
	}

	counts, e := cb.CounterGet(ctx, counterCol)
	if e != nil {
		t.Fatalf("CounterGet after commit: %v", e)
	}

	if counts["views"] != 5 {
		t.Fatalf("expected views=5 after committed increment, got %d", counts["views"])
	}

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if e := cb.CounterIncrement(ctx, counterCol, metaengine.Delta{"views": 3}); e != nil {
			return e
		}

		inside, e := cb.CounterGet(ctx, counterCol)
		if e != nil {
			return e
		}

		if inside["views"] != 8 {
			t.Errorf("expected views=8 inside tx, got %d", inside["views"])
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel from counter rollback, got %v", err)
	}

	counts, e = cb.CounterGet(ctx, counterCol)
	if e != nil {
		t.Fatalf("CounterGet after rollback: %v", e)
	}

	if counts["views"] != 5 {
		t.Fatalf("expected views=5 after rollback, got %d", counts["views"])
	}
}

func runStreamTxSubtest(
	t *testing.T, tx metaengine.Transactional, sb metaengine.StreamLogBackend,
	ctx context.Context, streamCol string, sentinel error,
) {
	t.Helper()
	streamID := "s1"

	if e := tx.RunInTx(ctx, func(ctx context.Context) error {
		return sb.StreamAppend(ctx, streamCol, streamID, []any{"a", "b"})
	}); e != nil {
		t.Fatalf("StreamAppend in tx (commit): %v", e)
	}

	values, e := sb.StreamRead(ctx, streamCol, streamID)
	if e != nil {
		t.Fatalf("StreamRead after commit: %v", e)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 values after commit, got %d", len(values))
	}

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if e := sb.StreamAppend(ctx, streamCol, streamID, []any{"c"}); e != nil {
			return e
		}

		inside, e := sb.StreamRead(ctx, streamCol, streamID)
		if e != nil {
			return e
		}

		if len(inside) != 3 {
			t.Errorf("expected 3 values inside tx, got %d", len(inside))
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel from stream rollback, got %v", err)
	}

	values, e = sb.StreamRead(ctx, streamCol, streamID)
	if e != nil {
		t.Fatalf("StreamRead after rollback: %v", e)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 values after rollback, got %d", len(values))
	}
}

func runMultimapTxSubtest(
	t *testing.T, tx metaengine.Transactional, mm metaengine.MultimapBackend,
	ctx context.Context, col string, sentinel error,
) {
	t.Helper()

	// Commit path: MultiAdd inside RunInTx, verify outside.
	if e := tx.RunInTx(ctx, func(ctx context.Context) error {
		return mm.MultiAdd(ctx, col, "k1", "v1")
	}); e != nil {
		t.Fatalf("MultiAdd in tx (commit): %v", e)
	}

	values, e := mm.MultiGet(ctx, col, "k1")
	if e != nil {
		t.Fatalf("MultiGet after commit: %v", e)
	}

	if len(values) != 1 {
		t.Fatalf("expected 1 value after commit, got %d", len(values))
	}

	// Rollback path: add a second value, return sentinel, verify it didn't persist.
	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if e := mm.MultiAdd(ctx, col, "k1", "v2"); e != nil {
			return e
		}

		inside, e := mm.MultiGet(ctx, col, "k1")
		if e != nil {
			return e
		}

		if len(inside) != 2 {
			t.Errorf("expected 2 values inside tx, got %d", len(inside))
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel from multimap rollback, got %v", err)
	}

	values, e = mm.MultiGet(ctx, col, "k1")
	if e != nil {
		t.Fatalf("MultiGet after rollback: %v", e)
	}

	if len(values) != 1 {
		t.Fatalf("expected 1 value after rollback, got %d", len(values))
	}
}

func runLogTxSubtest(
	t *testing.T, tx metaengine.Transactional, lb metaengine.LogBackend,
	ctx context.Context, col string, sentinel error,
) {
	t.Helper()

	// Commit path: LogAppend inside RunInTx, verify outside.
	if e := tx.RunInTx(ctx, func(ctx context.Context) error {
		return lb.LogAppend(ctx, col, "entry-1")
	}); e != nil {
		t.Fatalf("LogAppend in tx (commit): %v", e)
	}

	tail, e := lb.LogTail(ctx, col, 10)
	if e != nil {
		t.Fatalf("LogTail after commit: %v", e)
	}

	if len(tail) != 1 {
		t.Fatalf("expected 1 entry after commit, got %d", len(tail))
	}

	// Rollback path: append a second entry, return sentinel, verify it didn't persist.
	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if e := lb.LogAppend(ctx, col, "entry-2"); e != nil {
			return e
		}

		inside, e := lb.LogTail(ctx, col, 10)
		if e != nil {
			return e
		}

		if len(inside) != 2 {
			t.Errorf("expected 2 entries inside tx, got %d", len(inside))
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel from log rollback, got %v", err)
	}

	tail, e = lb.LogTail(ctx, col, 10)
	if e != nil {
		t.Fatalf("LogTail after rollback: %v", e)
	}

	if len(tail) != 1 {
		t.Fatalf("expected 1 entry after rollback, got %d", len(tail))
	}
}

// RunConcurrentTxTest verifies that concurrent RunInTx calls do not deadlock
// and that all committed writes are visible. Two goroutines each write a
// distinct key inside separate transactions; both must complete successfully.
//
// The engine must implement Transactional and MapBackend.
func RunConcurrentTxTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	tx, ok := eng.(metaengine.Transactional)
	if !ok {
		t.Fatalf("engine %T does not implement Transactional", eng)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("engine %T does not implement MapBackend", eng)
	}

	ctx := context.Background()
	col := "concurrent_tx_" + engineName(eng)

	var wg sync.WaitGroup

	errs := make([]error, 2)

	for i := range 2 {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			errs[idx] = tx.RunInTx(ctx, func(ctx context.Context) error {
				return mb.MapSet(ctx, col, key, fmt.Sprintf("val-%d", idx))
			})
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d RunInTx: %v", i, err)
		}
	}

	for i := range 2 {
		key := fmt.Sprintf("key-%d", i)
		_, found, err := mb.MapGet(ctx, col, key)
		if err != nil {
			t.Fatalf("MapGet %s: %v", key, err)
		}

		if !found {
			t.Fatalf("key %s not found after concurrent transactions", key)
		}
	}
}
