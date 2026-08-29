package mysqlengine

import (
	"context"
	"fmt"
	"hash/fnv"
)

// gcIndexPrefixLen is the prefix length for indexes on TEXT generated
// columns. MariaDB cannot index a full TEXT column; utf8mb4 at 190 chars
// keeps the composite (collection, gc(190)) under InnoDB's 3072-byte key
// limit (255*4 + 190*4 = 1780). Prefix indexes re-check the full column
// value after the index lookup, so filter semantics stay exact even for
// values longer than the prefix.
const gcIndexPrefixLen = 190

// gcColumnName derives a deterministic, collision-resistant column name for
// a JSON field's generated column: gc_<sanitized>_<fnv64-hex>. The hash
// suffix prevents two fields that sanitize identically (e.g. "a.b" and
// "a_b") from silently sharing one column with a different expression.
func gcColumnName(field string) string {
	return gcColumnNamePrefixed("gc_", field)
}

// gcNumColumnName derives the numeric twin of a generated column: the
// DECIMAL(65,10) cast of the field. Sort fields get both columns so ORDER BY
// can render (gcn, gc) — the column-based equivalent of the dual-key
// CAST/UNQUOTE sort — letting the composite index drive the sort.
func gcNumColumnName(field string) string {
	return gcColumnNamePrefixed("gcn_", field)
}

func gcColumnNamePrefixed(prefix, field string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(field))

	return fmt.Sprintf("%s%s_%08x", prefix, sanitizeIndexName(field), h.Sum64()&0xFFFFFFFF)
}

// applyMariaDBLayout implements ApplyLayout for the MariaDB dialect.
// MariaDB lacks MySQL 8 functional key parts, so filter/sort fields get a
// VIRTUAL generated column plus a plain composite index instead:
//
//	ALTER TABLE meta_map ADD COLUMN IF NOT EXISTS gc_<f>_<h> TEXT
//	  GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(value, '$.<f>'))) VIRTUAL
//	CREATE INDEX idx_map_gc_<f>_<h> ON meta_map (collection, gc_<f>_<h>(190))
//
// VIRTUAL keeps the ALTER metadata-only (no table rebuild — same mechanics
// as MySQL's hidden functional-index columns) and computes on read, so rows
// written before the layout was applied are covered with no backfill.
// MariaDB's optimizer does NOT substitute the generated column into raw
// JSON_UNQUOTE(JSON_EXTRACT(...)) predicates (verified on MariaDB 11.4),
// so PushdownMapScan renders gc-column references for laid-out fields via
// filterExpr; without that rewrite the index would be dead weight.
// Columns and indexes are shared per FIELD across collections: meta_map is
// one table, and the composite (collection, gc) index serves every
// collection, so duplicate identical indexes per collection are avoided.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) applyMariaDBLayout(filterFields, sortFields []string) error {
	seen := make(map[string]bool)

	sortFieldsSet := make(map[string]bool, len(sortFields))
	for _, f := range sortFields {
		sortFieldsSet[f] = true
	}

	allFields := append(append([]string{}, filterFields...), sortFields...)

	for _, field := range allFields {
		if seen[field] {
			continue
		}

		seen[field] = true

		if e.hasGcColumn(field) {
			// Field laid out earlier as a FILTER field: it has the text
			// column but no numeric twin. If it is now requested as a sort
			// field, add just the twin.
			if sortFieldsSet[field] && !e.hasGcNumColumn(field) {
				if err := e.applyMariaDBSortTwin(field, gcColumnName(field)); err != nil {
					return err
				}
			}

			continue
		}

		if err := e.applyMariaDBFieldColumn(field, sortFieldsSet[field]); err != nil {
			return err
		}
	}

	return nil
}

// applyMariaDBFieldColumn adds the TEXT generated column for one field, plus
// the numeric twin for sort fields, and their indexes.
func (e *mysqlEngine) applyMariaDBFieldColumn(field string, isSortField bool) error {
	column := gcColumnName(field)

	ddl := fmt.Sprintf(
		"ALTER TABLE meta_map ADD COLUMN IF NOT EXISTS %s TEXT "+
			"GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(value, '$.%s'))) VIRTUAL",
		column, escapeJSONPath(field),
	)

	if _, err := e.conn().ExecContext(context.Background(), ddl); err != nil {
		return fmt.Errorf("mysqlengine.ApplyLayout: add generated column %s: %w", column, err)
	}

	if isSortField {
		if err := e.applyMariaDBSortTwin(field, column); err != nil {
			return err
		}

		return nil
	}

	idxName := "idx_map_" + column

	idxDDL := fmt.Sprintf(
		"CREATE INDEX %s ON meta_map (collection, %s(%d))",
		idxName, column, gcIndexPrefixLen,
	)

	// MariaDB has no CREATE INDEX IF NOT EXISTS; a duplicate index name
	// (1061) means a previous run already created it.
	if _, err := e.conn().ExecContext(context.Background(), idxDDL); err != nil {
		if !isDuplicateIndexErr(err) {
			return fmt.Errorf("mysqlengine.ApplyLayout: create index %s: %w", idxName, err)
		}
	}

	e.recordGcColumn(field, column)

	return nil
}

// applyMariaDBSortTwin adds the DECIMAL numeric twin column and the
// composite (collection, gcn, gc) index. ORDER BY renders (gcn, gc): the
// column-based equivalent of the expression dual-key (CAST DECIMAL,
// JSON_UNQUOTE) — numerically for numbers, lexically for text — while the
// composite index lets the optimizer drive the sort instead of re-extracting
// JSON per row.
func (e *mysqlEngine) applyMariaDBSortTwin(field, textColumn string) error {
	numColumn := gcNumColumnName(field)

	ddl := fmt.Sprintf(
		"ALTER TABLE meta_map ADD COLUMN IF NOT EXISTS %s DECIMAL(65,10) "+
			"GENERATED ALWAYS AS (CAST(JSON_EXTRACT(value, '$.%s') AS DECIMAL(65,10))) VIRTUAL",
		numColumn, escapeJSONPath(field),
	)

	if _, err := e.conn().ExecContext(context.Background(), ddl); err != nil {
		return fmt.Errorf("mysqlengine.ApplyLayout: add numeric twin column %s: %w", numColumn, err)
	}

	idxName := "idx_map_sort_" + textColumn

	idxDDL := fmt.Sprintf(
		"CREATE INDEX %s ON meta_map (collection, %s, %s(%d))",
		idxName, numColumn, textColumn, gcIndexPrefixLen,
	)

	if _, err := e.conn().ExecContext(context.Background(), idxDDL); err != nil {
		if !isDuplicateIndexErr(err) {
			return fmt.Errorf("mysqlengine.ApplyLayout: create sort index %s: %w", idxName, err)
		}
	}

	e.recordGcColumn(field, textColumn)
	e.recordGcNumColumn(field, numColumn)

	return nil
}

// filterExpr renders the left-hand expression for a filter comparison,
// preferring the generated column created by ApplyLayout on MariaDB (which
// the composite index covers) over the raw JSON extraction expression.
// The generated column is defined as exactly JSON_UNQUOTE(JSON_EXTRACT(...)),
// and TEXT (not VARCHAR) avoids value truncation, so semantics match the
// expression form for missing fields (NULL → no match) and long values
// alike. Numeric parameters against the TEXT column coerce string-to-number
// exactly like the LONGTEXT expression did.
func (e *mysqlEngine) filterExpr(field string) string {
	if e.isMariaDB() {
		if m := e.gcColumns.Load(); m != nil {
			if column, ok := (*m)[field]; ok {
				return column
			}
		}
	}

	return e.jsonCompareExpr(field)
}

// hasGcNumColumn reports whether a numeric twin column was recorded for the
// field (sort fields only).
func (e *mysqlEngine) hasGcNumColumn(field string) bool {
	if m := e.gcnColumns.Load(); m != nil {
		_, ok := (*m)[field]

		return ok
	}

	return false
}

// recordGcNumColumn publishes the field→numeric-column map (copy-on-write,
// lock-free reads). Callers must hold layoutMu.
func (e *mysqlEngine) recordGcNumColumn(field, column string) {
	next := make(map[string]string, 1)
	if m := e.gcnColumns.Load(); m != nil {
		for k, v := range *m {
			next[k] = v
		}
	}

	next[field] = column
	e.gcnColumns.Store(&next)
}

// gcNumColumnFor returns the numeric twin column for the field, if any.
func (e *mysqlEngine) gcNumColumnFor(field string) (string, bool) {
	if m := e.gcnColumns.Load(); m != nil {
		column, ok := (*m)[field]

		return column, ok
	}

	return "", false
}

// hasGcColumn reports whether a generated column was already recorded for
// the field in this engine instance.
func (e *mysqlEngine) hasGcColumn(field string) bool {
	if m := e.gcColumns.Load(); m != nil {
		_, ok := (*m)[field]

		return ok
	}

	return false
}

// recordGcColumn publishes field→column in the copy-on-write map so query
// paths read it lock-free. Callers must hold layoutMu.
func (e *mysqlEngine) recordGcColumn(field, column string) {
	next := make(map[string]string, 1)
	if m := e.gcColumns.Load(); m != nil {
		for k, v := range *m {
			next[k] = v
		}
	}

	next[field] = column
	e.gcColumns.Store(&next)
}
