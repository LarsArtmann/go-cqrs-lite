package indexing

import (
	"testing"
)

func TestParseStat1Rows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		stat string
		want int64
	}{
		{name: "simple", stat: "rows=12345", want: 12345},
		{name: "with_extra_fields", stat: "rows=999 width=3 height=4", want: 999},
		{name: "zero", stat: "rows=0", want: 0},
		{name: "missing_prefix", stat: "width=3 height=4", want: 0},
		{name: "empty", stat: "", want: 0},
		{name: "negative_value_not_parsed", stat: "rows=-1 width=2", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseStat1Rows(tt.stat)
			if got != tt.want {
				t.Errorf("parseStat1Rows(%q) = %d, want %d", tt.stat, got, tt.want)
			}
		})
	}
}

func TestInferIndex_EventsAggregateVersion(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, expl, prio := a.inferIndex("events",
		"SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ? AND version > ?")

	if idx == nil {
		t.Fatal("expected index recommendation, got nil")
	}

	if idx.Name != "idx_events_agg_ver" {
		t.Errorf("index name: got %s, want idx_events_agg_ver", idx.Name)
	}

	if prio != PriorityCritical {
		t.Errorf("priority: got %s, want %s", prio, PriorityCritical)
	}

	if expl == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestInferIndex_EventsEventType(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, prio := a.inferIndex("events",
		"SELECT * FROM events WHERE event_type = ?")

	if idx == nil {
		t.Fatal("expected index recommendation, got nil")
	}

	if idx.Name != "idx_events_type_time" {
		t.Errorf("index name: got %s, want idx_events_type_time", idx.Name)
	}

	if prio != PriorityRecommended {
		t.Errorf("priority: got %s, want %s", prio, PriorityRecommended)
	}
}

func TestInferIndex_EventsCursor(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, prio := a.inferIndex("events",
		"SELECT * FROM events WHERE occurred_at > ? AND id > ? ORDER BY occurred_at")

	if idx == nil {
		t.Fatal("expected index recommendation, got nil")
	}

	if idx.Name != "idx_events_cursor" {
		t.Errorf("index name: got %s, want idx_events_cursor", idx.Name)
	}

	if prio != PriorityCritical {
		t.Errorf("priority: got %s, want %s", prio, PriorityCritical)
	}
}

func TestInferIndex_CommandsAggregate(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, prio := a.inferIndex("commands",
		"SELECT * FROM commands WHERE aggregate_type = ? AND aggregate_id = ?")

	if idx == nil {
		t.Fatal("expected index recommendation, got nil")
	}

	if idx.Name != "idx_commands_agg_time" {
		t.Errorf("index name: got %s, want idx_commands_agg_time", idx.Name)
	}

	if prio != PriorityRecommended {
		t.Errorf("priority: got %s, want %s", prio, PriorityRecommended)
	}
}

func TestInferIndex_CommandsType(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, prio := a.inferIndex("commands",
		"SELECT * FROM commands WHERE command_type = ?")

	if idx == nil {
		t.Fatal("expected index recommendation, got nil")
	}

	if idx.Name != "idx_commands_type_time" {
		t.Errorf("index name: got %s, want idx_commands_type_time", idx.Name)
	}

	if prio != PriorityOptional {
		t.Errorf("priority: got %s, want %s", prio, PriorityOptional)
	}
}

func TestInferIndex_UnknownTable(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, _ := a.inferIndex("unknown_table",
		"SELECT * FROM unknown_table WHERE foo = ?")

	if idx != nil {
		t.Errorf("expected nil for unknown table, got %+v", idx)
	}
}

func TestInferIndex_NoMatch(t *testing.T) {
	t.Parallel()

	a := &Advisor{}
	idx, _, _ := a.inferIndex("events",
		"SELECT * FROM events WHERE unrelated_column = ?")

	if idx != nil {
		t.Errorf("expected nil for unmatched query, got %+v", idx)
	}
}

func TestScanTableRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		detail string
		table  string
		match  bool
	}{
		{name: "modern_format", detail: "SCAN events", table: "events", match: true},
		{name: "legacy_format", detail: "SCAN TABLE events", table: "events", match: true},
		{name: "modern_with_alias", detail: "SCAN events AS e", table: "events", match: true},
		{name: "search_not_scan", detail: "SEARCH events USING INDEX", table: "", match: false},
		{name: "empty", detail: "", table: "", match: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := scanTableRe.FindStringSubmatch(tt.detail)
			if tt.match {
				if m == nil {
					t.Errorf("expected match for %q", tt.detail)
					return
				}
				if m[1] != tt.table {
					t.Errorf("table: got %q, want %q", m[1], tt.table)
				}
			} else if m != nil {
				t.Errorf("expected no match for %q, got table %q", tt.detail, m[1])
			}
		})
	}
}
