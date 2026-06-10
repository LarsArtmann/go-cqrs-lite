package indexing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
)

func TestIndex_DDL(t *testing.T) {
	t.Parallel()

	idx := indexing.Index{
		Name:    "idx_test",
		Table:   "events",
		Columns: []string{"aggregate_type", "aggregate_id"},
	}

	want := "CREATE INDEX IF NOT EXISTS idx_test ON events(aggregate_type, aggregate_id);"
	if got := idx.DDL(); got != want {
		t.Errorf("DDL() = %q, want %q", got, want)
	}
}

func TestIndex_DDL_unique(t *testing.T) {
	t.Parallel()

	idx := indexing.Index{
		Name:    "idx_test_unique",
		Table:   "events",
		Columns: []string{"id"},
		Unique:  true,
	}

	want := "CREATE UNIQUE INDEX IF NOT EXISTS idx_test_unique ON events(id);"
	if got := idx.DDL(); got != want {
		t.Errorf("DDL() = %q, want %q", got, want)
	}
}

func TestIndex_DDL_partial(t *testing.T) {
	t.Parallel()

	idx := indexing.Index{
		Name:    "idx_test_partial",
		Table:   "events",
		Columns: []string{"event_type"},
		Where:   "schema_version > 1",
	}

	want := "CREATE INDEX IF NOT EXISTS idx_test_partial ON events(event_type) WHERE schema_version > 1;"
	if got := idx.DDL(); got != want {
		t.Errorf("DDL() = %q, want %q", got, want)
	}
}

func TestIndex_DropDDL(t *testing.T) {
	t.Parallel()

	idx := indexing.Index{Name: "idx_test", Table: "events", Columns: []string{"id"}}

	want := "DROP INDEX IF EXISTS idx_test;"
	if got := idx.DropDDL(); got != want {
		t.Errorf("DropDDL() = %q, want %q", got, want)
	}
}

func TestIndexSet_DDL(t *testing.T) {
	t.Parallel()

	set := indexing.IndexSet{
		{Name: "idx_a", Table: "events", Columns: []string{"a"}},
		{Name: "idx_b", Table: "events", Columns: []string{"b"}},
	}

	ddls := set.DDL()
	if len(ddls) != 2 {
		t.Fatalf("expected 2 DDLs, got %d", len(ddls))
	}
}

func TestIndexSet_Filter(t *testing.T) {
	t.Parallel()

	set := indexing.IndexSet{
		{Name: "idx_events", Table: "events", Columns: []string{"a"}},
		{Name: "idx_commands", Table: "commands", Columns: []string{"b"}},
	}

	filtered := set.Filter("events")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered index, got %d", len(filtered))
	}

	if filtered[0].Name != "idx_events" {
		t.Errorf("Name = %q, want idx_events", filtered[0].Name)
	}
}

func TestIndexSet_Names(t *testing.T) {
	t.Parallel()

	set := indexing.IndexSet{
		{Name: "idx_a", Table: "t", Columns: []string{"a"}},
		{Name: "idx_b", Table: "t", Columns: []string{"b"}},
	}

	names := set.Names()
	if len(names) != 2 || names[0] != "idx_a" || names[1] != "idx_b" {
		t.Errorf("Names = %v, want [idx_a idx_b]", names)
	}
}

func TestRecommendedCQRSIndexes(t *testing.T) {
	t.Parallel()

	idxs := indexing.RecommendedCQRSIndexes()
	if len(idxs) == 0 {
		t.Fatal("expected non-empty recommended indexes")
	}

	tables := make(map[string]int)
	for _, idx := range idxs {
		tables[idx.Table]++
	}

	if tables["events"] == 0 {
		t.Error("expected events indexes")
	}

	if tables["commands"] == 0 {
		t.Error("expected commands indexes")
	}
}
