package bench_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// multi_engine_integration_test.go exercises the runtime role lifecycle
// (ADR-0124 §7) across two REAL backends — SQLite (Row) and Pebble (LSM),
// both CGo-free — instead of memory engines:
//
//	plan → apply → AddEngine(Migration) → Backfill → PromoteEngine →
//	DemoteEngine → live mirroring → both engines serve identical results.
//
// Folds are intentionally idempotent (insert): Backfill and the demote
// re-routed replay then land exactly-once-by-overwrite on every engine that
// already holds the collection (e.g. an engine that mirrored it earlier).

type miItemCreated struct {
	ID    string
	Title string
}

type miItem struct {
	Title string `json:"Title"`
}

type miFindItem struct {
	ID string
}

func miItemQuery() any {
	return metaengine.Query[miFindItem, miItem](
		"mi_items",
		metaengine.OnRecord(
			miItemCreated{},
			func(_ record.Record, e miItemCreated) (string, miItem) {
				return e.ID, miItem{Title: e.Title}
			},
		),
	)
}

func miApply(t *testing.T, store *metaengine.Store, id string) {
	t.Helper()

	if err := store.Apply(context.Background(), "miItemCreated", miItemCreated{
		ID: id, Title: "title-" + id,
	}); err != nil {
		t.Fatalf("apply %s: %v", id, err)
	}
}

func miGet(t *testing.T, eng metaengine.Engine, id string) (miItem, bool) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("%T is not a MapBackend", eng)
	}

	raw, found, err := mb.MapGet(context.Background(), "mi_items", id)
	if err != nil {
		t.Fatalf("MapGet %s on %s: %v", id, eng.Profile().Name, err)
	}

	if !found {
		return miItem{}, false
	}

	switch v := raw.(type) {
	case miItem:
		return v, true
	case map[string]any:
		title, _ := v["Title"].(string)
		if title == "" {
			title, _ = v["title"].(string)
		}

		return miItem{Title: title}, true
	default:
		t.Fatalf("MapGet %s on %s returned %T", id, eng.Profile().Name, raw)

		return miItem{}, false
	}
}

func miAssertItem(t *testing.T, eng metaengine.Engine, id string) {
	t.Helper()

	item, found := miGet(t, eng, id)
	if !found || item.Title != "title-"+id {
		t.Fatalf(
			"engine %s: item %s = %#v (found=%v), want title-%s",
			eng.Profile().Name, id, item, found, id,
		)
	}
}

func miWaitFor(t *testing.T, d time.Duration, cond func() bool, what string) {
	t.Helper()

	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// miExecute asserts the store serves the item with the expected title. Row
// engines implement RawValueReader, so Execute returns metaengine.JSONValue;
// memory-shaped engines return the decoded value directly.
func miExecute(t *testing.T, store *metaengine.Store, id string) {
	t.Helper()

	res, err := store.Execute(miFindItem{ID: id})
	if err != nil {
		t.Fatalf("execute %s: %v", id, err)
	}

	var item miItem

	switch v := res.(type) {
	case metaengine.JSONValue:
		if err := json.Unmarshal(v, &item); err != nil {
			t.Fatalf("execute %s: decode %T: %v", id, res, err)
		}
	case miItem:
		item = v
	default:
		t.Fatalf("execute %s returned %T", id, res)
	}

	if item.Title != "title-"+id {
		t.Fatalf("execute %s = %#v, want title-%s", id, item, id)
	}
}

// TestMultiEngine_TwoRealBackends_CutoverLifecycle is the integration test the
// Phase 6b TODO asked for: two live engines with data, AddEngine + Backfill,
// both serve correct query results — extended through the promote/demote
// lifecycle so every role transition is proven on real storage.
func TestMultiEngine_TwoRealBackends_CutoverLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sqliteEng, sdb := newSQLiteEngine()
	t.Cleanup(func() { _ = sdb.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{sqliteEng}, miItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	metaengine.WithEventLog(store, metaengine.NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	for i := range 5 {
		miApply(t, store, "mi-"+string(rune('0'+i+1)))
	}

	miExecute(t, store, "mi-3")

	miAssertItem(t, sqliteEng, "mi-3")

	// Cutover runbook (METAENGINE-LAYOUT-ROLES §4.2) on a real LSM engine.
	pebbleEng := newPebbleEngine(t)

	if err := store.AddEngine(
		ctx,
		pebbleEng,
		metaengine.WithEngineRole(metaengine.RoleMigration),
	); err != nil {
		t.Fatal(err)
	}

	if st, ok := store.ReplicationStatus("pebble"); !ok || st.Role != metaengine.RoleMigration {
		t.Fatalf("replication status after add = %#v ok=%v", st, ok)
	}

	// Idempotent folds: plain Backfill is safe — no WithBackfillForce.
	if err := store.Backfill(ctx); err != nil {
		t.Fatal(err)
	}

	// BOTH live backends now serve identical, correct results.
	for i := range 5 {
		miAssertItem(t, sqliteEng, "mi-"+string(rune('0'+i+1)))
		miAssertItem(t, pebbleEng, "mi-"+string(rune('0'+i+1)))
	}

	// Live traffic mirrors while in Migration role.
	miApply(t, store, "mi-6")

	miWaitFor(t, 2*time.Second, func() bool {
		_, found := miGet(t, pebbleEng, "mi-6")

		return found
	}, "mi-6 to replicate to the migration engine")

	if err := store.PromoteEngine(ctx, "pebble"); err != nil {
		t.Fatal(err)
	}

	if role, _ := store.EngineRole("pebble"); role != metaengine.RoleActive {
		t.Fatalf("pebble role after promote = %q", role)
	}

	// Post-promote exactly one engine receives primary folds for the query;
	// demote the other so it keeps serving as a mirror (and the promoted
	// engine takes over if it had not already).
	miApply(t, store, "mi-7")

	miExecute(t, store, "mi-7")

	assigned := store.Plan().Queries[0].EngineName

	var demotedName string

	if assigned == "pebble" {
		demotedName = "sqlite"
	} else {
		demotedName = "pebble"
	}

	demoted := sqliteEng
	if demotedName == "pebble" {
		demoted = pebbleEng
	}

	if err := store.DemoteEngine(ctx, demotedName); err != nil {
		t.Fatalf("demote %s: %v", demotedName, err)
	}

	if role, _ := store.EngineRole(demotedName); role != metaengine.RoleBackup {
		t.Fatalf("%s role after demote = %q", demotedName, role)
	}

	for _, qa := range store.Plan().Queries {
		if qa.EngineName == demotedName {
			t.Fatalf("query %q still routed to demoted %s", qa.QueryName, demotedName)
		}
	}

	// Live traffic after demotion: primary serves, mirror stays current.
	miApply(t, store, "mi-8")

	miWaitFor(t, 2*time.Second, func() bool {
		_, found := miGet(t, demoted, "mi-8")

		return found
	}, "mi-8 to replicate to the demoted mirror")

	miExecute(t, store, "mi-8")

	// Final contract: every live backend serves the complete, identical state.
	for i := 1; i <= 8; i++ {
		miAssertItem(t, sqliteEng, "mi-"+string(rune('0'+i)))
		miAssertItem(t, pebbleEng, "mi-"+string(rune('0'+i)))
	}
}

// TestMultiEngine_BackfillRefusesNonIdempotent proves the Backfill safety
// guard fires on real engines too: a counter fold blocks replay unless the
// operator forces it.
func TestMultiEngine_BackfillRefusesNonIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sqliteEng, sdb := newSQLiteEngine()
	t.Cleanup(func() { _ = sdb.Close() })

	store, err := metaengine.Plan(
		[]metaengine.Engine{sqliteEng},
		metaengine.Query[struct{}, map[string]int64](
			"mi_counts",
			metaengine.OnRecord(
				miItemCreated{},
				func(_ record.Record, _ miItemCreated) metaengine.Delta {
					return metaengine.Delta{"created": +1}
				},
			),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	metaengine.WithEventLog(store, metaengine.NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	miApply(t, store, "mi-1")

	if err := store.AddEngine(
		ctx,
		newPebbleEngine(t),
		metaengine.WithEngineRole(metaengine.RoleMigration),
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Backfill(ctx); err == nil {
		t.Fatal("Backfill must refuse non-idempotent counter folds on live engines")
	}

	// Force on a live primary is the operator's risk; here the fresh mirror is
	// the only empty projection and the counter on sqlite doubles — which is
	// exactly why the guard exists. Do not assert values after forcing.
}
