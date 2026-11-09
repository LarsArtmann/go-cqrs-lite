package indexing

import (
	"fmt"
	"strings"
)

// Index describes a single database index.
type Index struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
	Partial bool   // true if this is a partial index (WHERE clause)
	Where   string // partial-index predicate; only used when Partial is true
	Reason  string // human-readable justification
}

// DDL returns the CREATE INDEX statement for this index.
func (i Index) DDL() string {
	var builder strings.Builder

	if i.Unique {
		builder.WriteString("CREATE UNIQUE INDEX IF NOT EXISTS ")
	} else {
		builder.WriteString("CREATE INDEX IF NOT EXISTS ")
	}

	builder.WriteString(i.Name)
	builder.WriteString(" ON ")
	builder.WriteString(i.Table)
	builder.WriteString("(")
	builder.WriteString(strings.Join(i.Columns, ", "))
	builder.WriteString(")")

	if i.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(i.Where)
	}

	builder.WriteString(";")

	return builder.String()
}

// DropDDL returns the DROP INDEX statement.
func (i Index) DropDDL() string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s;", i.Name)
}

// IndexSet is a collection of indexes.
type IndexSet []Index

// DDL returns all CREATE INDEX statements.
func (s IndexSet) DDL() []string {
	ddls := make([]string, len(s))
	for j, idx := range s {
		ddls[j] = idx.DDL()
	}

	return ddls
}

// Filter returns indexes for a specific table.
func (s IndexSet) Filter(table string) IndexSet {
	var filtered IndexSet

	for _, idx := range s {
		if idx.Table == table {
			filtered = append(filtered, idx)
		}
	}

	return filtered
}

// Names returns the names of all indexes in the set.
func (s IndexSet) Names() []string {
	names := make([]string, len(s))
	for j, idx := range s {
		names[j] = idx.Name
	}

	return names
}

// DropDDL returns all DROP INDEX statements for the set.
func (s IndexSet) DropDDL() []string {
	ddls := make([]string, len(s))
	for j, idx := range s {
		ddls[j] = idx.DropDDL()
	}

	return ddls
}

// RecommendedCQRSIndexes returns pre-calculated indexes optimized for
// common CQRS event-sourcing access patterns on a Turso/SQLite schema.
func RecommendedCQRSIndexes() IndexSet {
	return IndexSet{
		{
			Name:    "idx_events_cursor",
			Table:   "events",
			Columns: []string{"occurred_at", "id"},
			Reason:  "cursor pagination for ReadFrom / journal replay",
		},
		{
			Name:    "idx_events_agg_ver",
			Table:   "events",
			Columns: []string{"aggregate_type", "aggregate_id", "version"},
			Reason:  "covering index for LoadFromVersion / LoadToVersion",
		},
		{
			Name:    "idx_events_type_time",
			Table:   "events",
			Columns: []string{"event_type", "occurred_at"},
			Reason:  "projection filters by event type with ordering",
		},
		{
			Name:    "idx_commands_agg_time",
			Table:   "commands",
			Columns: []string{"aggregate_type", "aggregate_id", "received_at"},
			Reason:  "command audit trail with time ordering",
		},
		{
			Name:    "idx_commands_type_time",
			Table:   "commands",
			Columns: []string{"command_type", "received_at"},
			Reason:  "command type analytics",
		},
	}
}
