package sqliteengine

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mapSortIndex is one operator-declared expression index over meta_map.
type mapSortIndex struct {
	collection string
	path       string
	desc       bool
	name       string
}

// WithSortIndex declares an opt-in expression index over a meta_map
// collection's JSON field, accelerating the standard (unplanned) sorted and
// keyset-paginated scans that order by json_extract(value, path):
//
//	eng, err := sqliteengine.NewSQLiteEngine(db,
//		sqliteengine.WithSortIndex("captures", "$.last_seen", true))
//
// NewSQLiteEngine emits the same CREATE INDEX IF NOT EXISTS form the journal
// already carries (idx_stream_log_event_id), partial on json_valid(value):
//
//	CREATE INDEX IF NOT EXISTS idx_map_sort_<collection>_<field>
//	  ON meta_map(collection, json_extract(value, '$.last_seen') DESC)
//	  WHERE json_valid(value);
//
// Measured on a 100k-row store, a 61-row keyset page on the indexed sort
// column drops from ~58ms (full-collection scan + temp B-tree) to ~0.2ms
// (issue #122). The index only accelerates STANDARD meta_map scans — a
// collection migrated to a planned table (ApplyLayout) ignores it, and the
// option keeps no state that would break that migration.
//
// desc must match the direction the scans order by (ORDER BY expr DESC uses
// a DESC index directly; ASC scans want desc=false). Every row rewrite
// maintains the index — the write-amplitude trade is why this is opt-in per
// collection instead of a global default. Call the option multiple times to
// index several collections or fields.
func WithSortIndex(collection, path string, desc bool) EngineOption {
	return func(e *sqliteEngine) {
		// Normalize to the exact expression the standard scans order by
		// (jsonPath prefixes "$."): an index only helps when its expression
		// is byte-identical to the query's json_extract argument.
		if !strings.HasPrefix(path, "$") {
			path = "$." + path
		}

		idx := mapSortIndex{
			collection: collection,
			path:       path,
			desc:       desc,
			name:       mapSortIndexName(collection, path),
		}

		if err := idx.validate(); err != nil {
			e.sortIndexErr = err

			return
		}

		e.sortIndexes = append(e.sortIndexes, idx)
	}
}

// validate rejects declarations that cannot produce a usable index: empty
// collection, or a field reference that is not a JSON path after
// normalization ("$.field").
func (idx mapSortIndex) validate() error {
	if idx.collection == "" {
		return fmt.Errorf("sqliteengine: WithSortIndex: collection must not be empty")
	}

	if !strings.HasPrefix(idx.path, "$.") || len(idx.path) <= 2 {
		return fmt.Errorf(
			"sqliteengine: WithSortIndex: path %q must name a field (e.g. \"$.last_seen\" or \"last_seen\")",
			idx.path,
		)
	}

	return nil
}

// mapSortIndexName derives a stable index name from the declaration:
// non-identifier runes collapse to underscores, so "captures" + "$.last_seen"
// becomes idx_map_sort_captures_last_seen.
func mapSortIndexName(collection, path string) string {
	sanitize := func(s string) string {
		var b strings.Builder

		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				b.WriteRune(r)

				continue
			}

			b.WriteByte('_')
		}

		return b.String()
	}

	return "idx_map_sort_" + sanitize(collection) + "_" + sanitize(path)
}

// ddl renders the CREATE INDEX statement. The trailing key column mirrors
// the scans' ORDER BY tiebreak (expr, then key ASC) so the index fully
// satisfies the ordering — without it the planner keeps the PK scan plus a
// temp B-tree. Unlike the journal's idx_stream_log_event_id, the index is
// NOT partial: a partial json_valid(value) index is unusable for these
// scans (their WHERE does not imply the predicate), and meta_map values are
// engine-encoded JSON by construction, so the expression never sees
// non-JSON rows.
func (idx mapSortIndex) ddl() string {
	dir := ""
	if idx.desc {
		dir = " DESC"
	}

	// Escape single quotes exactly like jsonPath does, so the index
	// expression stays byte-identical to the scans' json_extract argument.
	esc := strings.ReplaceAll(idx.path, "'", "''")

	return fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON meta_map(collection, json_extract(value, '%s')%s, key)",
		metaengine.QuoteIdent(idx.name),
		esc,
		dir,
	)
}

// createSortIndexes executes every declared index after the base tables
// exist. IF NOT EXISTS makes re-construction idempotent; a changed
// declaration under the same derived name is a silent no-op by SQLite's
// rules (drop the old index to re-declare).
func (e *sqliteEngine) createSortIndexes(ctx context.Context) error {
	if e.sortIndexErr != nil {
		return fmt.Errorf("metaengine: sort index: %w", e.sortIndexErr)
	}

	for _, idx := range e.sortIndexes {
		if _, err := e.db.ExecContext(ctx, idx.ddl()); err != nil {
			return fmt.Errorf("metaengine: create sort index %s: %w", idx.name, err)
		}
	}

	return nil
}
