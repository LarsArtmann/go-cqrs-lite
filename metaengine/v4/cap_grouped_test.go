package metaengine_test

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type groupedEvent struct {
	ID    string
	Owner string
	Seq   int
}

type groupedItem struct {
	ID    string
	Owner string
	Seq   int
}

func groupedQuery(name string, maxRows, perGroup int) metaengine.QueryDecl[struct{}, groupedItem] {
	return metaengine.Query[struct{}, groupedItem](name,
		metaengine.OnRecord(groupedEvent{}, func(_ record.Record, e groupedEvent) (string, groupedItem) {
			return e.ID, groupedItem(e)
		}),
		metaengine.WithCap(maxRows, "Seq"),
		metaengine.WithCapPerGroup("Owner", perGroup),
	)
}

func newGroupedStore(t *testing.T, query any) *metaengine.Store {
	t.Helper()

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	store, err := metaengine.Plan([]metaengine.Engine{mem}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

// The portable contract on the memory engine: the Go fallback path (no
// CapGroupTrimmer pushdown) keeps the newest rows per group, groups are
// isolated from each other's floods, and the trim is idempotent.
func TestWithCapPerGroup_MemoryFallbackContract(t *testing.T) {
	t.Parallel()

	adttest.AssertGroupedCapContract(t, metaengine.NewMemoryEngine())
}

// EnforceCapGrouped trims ONLY the grouped bound: the collection-global cap
// is deliberately not applied by the grouped call (EnforceCap composes both),
// so here a large global cap leaves the total untouched except for the one
// owner over its per-group quota.
func TestWithCapPerGroup_EnforceCapGroupedStandalone(t *testing.T) {
	t.Parallel()

	store := newGroupedStore(t, groupedQuery("grouped_standalone", 100, 2))
	ctx := t.Context()

	seed := []groupedEvent{
		{ID: "a1", Owner: "a", Seq: 1},
		{ID: "a2", Owner: "a", Seq: 2},
		{ID: "a3", Owner: "a", Seq: 3},
		{ID: "b1", Owner: "b", Seq: 1},
	}

	for _, evt := range seed {
		if err := store.Apply(ctx, "groupedEvent", evt); err != nil {
			t.Fatalf("Apply %s: %v", evt.ID, err)
		}
	}

	removed, err := store.EnforceCapGrouped(ctx, "grouped_standalone")
	if err != nil {
		t.Fatalf("EnforceCapGrouped: %v", err)
	}

	if removed != 1 {
		t.Fatalf("EnforceCapGrouped removed %d rows, want 1 (a keeps Seq 2,3)", removed)
	}

	reader := metaengine.NewReader[groupedItem](store, "grouped_standalone")

	rows, err := reader.Scan(ctx, metaengine.WithLimit(0), metaengine.WithSort("Seq", false))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("after grouped trim: %d rows, want 3 (global cap 100 untouched)", len(rows))
	}

	if rows[0].ID != "b1" || rows[1].ID != "a2" || rows[2].ID != "a3" {
		t.Fatalf("survivors %v, want b1,a2,a3 ascending (a keeps its newest 2)", rows)
	}
}

// EnforceCapGrouped on a query without a grouped declaration is a no-op, and
// an unknown query name is the standard lookup error.
func TestWithCapPerGroup_StandaloneDegenerateCases(t *testing.T) {
	t.Parallel()

	store := newCappedStore(t, cappedQuery(3, "Seq"))
	seedCapEvents(t, store, 6)

	removed, err := store.EnforceCapGrouped(t.Context(), "recent_items")
	if err != nil {
		t.Fatalf("EnforceCapGrouped on non-grouped query: %v", err)
	}

	if removed != 0 {
		t.Fatalf("non-grouped EnforceCapGrouped removed %d rows, want 0", removed)
	}

	if _, err := store.EnforceCapGrouped(t.Context(), "no_such_query"); err == nil {
		t.Fatal("unknown query name: want error, got nil")
	}
}

// EnforceCaps reaches grouped queries too (they declare WithCap, so they are
// in the capped set; EnforceCap applies grouped + global composition).
func TestWithCapPerGroup_EnforceCapsCoversGrouped(t *testing.T) {
	t.Parallel()

	store := newGroupedStore(t, groupedQuery("grouped_caps_pass", 4, 2))
	ctx := t.Context()

	seed := []groupedEvent{
		{ID: "a1", Owner: "a", Seq: 1},
		{ID: "a2", Owner: "a", Seq: 2},
		{ID: "a3", Owner: "a", Seq: 3},
		{ID: "b1", Owner: "b", Seq: 1},
		{ID: "b2", Owner: "b", Seq: 2},
		{ID: "b3", Owner: "b", Seq: 3},
	}

	for _, evt := range seed {
		if err := store.Apply(ctx, "groupedEvent", evt); err != nil {
			t.Fatalf("Apply %s: %v", evt.ID, err)
		}
	}

	total, err := store.EnforceCaps(ctx)
	if err != nil {
		t.Fatalf("EnforceCaps: %v", err)
	}

	// Grouped pass removes a1 and b1 (each owner keeps its newest 2); the
	// global cap of 4 then holds exactly (a2,a3,b2,b3), so nothing more goes.
	if total != 2 {
		t.Fatalf("EnforceCaps removed %d rows, want 2", total)
	}
}

// The config accessors round-trip the declaration; zero values mean
// collection-global.
func TestWithCapPerGroup_ConfigAccessors(t *testing.T) {
	t.Parallel()

	cfg := groupedQuery("grouped_accessors", 8, 2).QueryConfig()
	if cfg.CapGroupCol() != "Owner" || cfg.CapGroupRows() != 2 {
		t.Fatalf("grouped accessors: col %q rows %d, want Owner/2", cfg.CapGroupCol(), cfg.CapGroupRows())
	}

	if cfg.CapRows() != 8 || cfg.CapOrderBy() != "Seq" {
		t.Fatalf("base cap accessors: rows %d order %q, want 8/Seq", cfg.CapRows(), cfg.CapOrderBy())
	}

	plain := cappedQuery(3, "Seq").QueryConfig()
	if plain.CapGroupCol() != "" || plain.CapGroupRows() != 0 {
		t.Fatalf("plain cap: group col %q rows %d, want zero values", plain.CapGroupCol(), plain.CapGroupRows())
	}
}

// A grouped declaration without its WithCap base is rejected at Plan time.
func TestWithCapPerGroup_RejectsMissingWithCap(t *testing.T) {
	t.Parallel()

	query := metaengine.Query[struct{}, groupedItem]("grouped_orphan",
		metaengine.OnRecord(groupedEvent{}, func(_ record.Record, e groupedEvent) (string, groupedItem) {
			return e.ID, groupedItem(e)
		}),
		metaengine.WithCapPerGroup("Owner", 2),
	)

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	_, err := metaengine.Plan([]metaengine.Engine{mem}, query)
	if err == nil {
		t.Fatal("Plan with grouped cap but no WithCap: want error, got nil")
	}

	if !strings.Contains(err.Error(), "requires a WithCap declaration") {
		t.Fatalf("error %v, want requires-WithCap message", err)
	}
}

// Group columns that are not plain identifiers are rejected at declaration
// time — they flow into engine SQL text, so anything else is a build-time
// error, not a runtime injection surface.
func TestWithCapPerGroup_RejectsNonIdentifierColumns(t *testing.T) {
	t.Parallel()

	for _, groupCol := range []string{"Owner; DROP TABLE x", "owner name", "", "1owner"} {
		query := metaengine.Query[struct{}, groupedItem]("grouped_bad_ident",
			metaengine.OnRecord(groupedEvent{}, func(_ record.Record, e groupedEvent) (string, groupedItem) {
				return e.ID, groupedItem(e)
			}),
			metaengine.WithCap(8, "Seq"),
			metaengine.WithCapPerGroup(groupCol, 2),
		)

		mem := metaengine.NewMemoryEngine()

		_, err := metaengine.Plan([]metaengine.Engine{mem}, query)
		_ = mem.Close()

		if err == nil {
			t.Fatalf("Plan with group column %q: want error, got nil", groupCol)
		}

		if !strings.Contains(err.Error(), "plain identifiers") {
			t.Fatalf("group column %q: error %v, want plain-identifier message", groupCol, err)
		}
	}
}

// A group column with no matching field on the result type (name or json tag,
// case-insensitive) is rejected at declaration time.
func TestWithCapPerGroup_RejectsUnknownGroupColumn(t *testing.T) {
	t.Parallel()

	query := metaengine.Query[struct{}, groupedItem]("grouped_bad_column",
		metaengine.OnRecord(groupedEvent{}, func(_ record.Record, e groupedEvent) (string, groupedItem) {
			return e.ID, groupedItem(e)
		}),
		metaengine.WithCap(8, "Seq"),
		metaengine.WithCapPerGroup("Tenant", 2),
	)

	mem := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = mem.Close() })

	_, err := metaengine.Plan([]metaengine.Engine{mem}, query)
	if err == nil {
		t.Fatal("Plan with unknown group column: want error, got nil")
	}

	if !strings.Contains(err.Error(), "not found on the result type") {
		t.Fatalf("error %v, want column-not-found message", err)
	}
}

// The group column may match a json tag instead of the field name, and the
// match is case-insensitive — the same matching ExtractFields applies.
func TestWithCapPerGroup_MatchesJSONTagCaseInsensitively(t *testing.T) {
	t.Parallel()

	type taggedItem struct {
		ID    string
		Owner string `json:"tenant_id"`
		Seq   int
	}

	query := metaengine.Query[struct{}, taggedItem]("grouped_tagged",
		metaengine.OnRecord(groupedEvent{}, func(_ record.Record, e groupedEvent) (string, taggedItem) {
			return e.ID, taggedItem(e)
		}),
		metaengine.WithCap(8, "Seq"),
		metaengine.WithCapPerGroup("TENANT_ID", 2),
	)

	store := newGroupedStore(t, query)
	ctx := t.Context()

	for _, evt := range []groupedEvent{
		{ID: "a1", Owner: "x", Seq: 1},
		{ID: "a2", Owner: "x", Seq: 2},
		{ID: "a3", Owner: "x", Seq: 3},
	} {
		if err := store.Apply(ctx, "groupedEvent", evt); err != nil {
			t.Fatalf("Apply %s: %v", evt.ID, err)
		}
	}

	removed, err := store.EnforceCapGrouped(ctx, "grouped_tagged")
	if err != nil {
		t.Fatalf("EnforceCapGrouped: %v", err)
	}

	if removed != 1 {
		t.Fatalf("removed %d rows, want 1 (owner x keeps Seq 2,3)", removed)
	}
}
