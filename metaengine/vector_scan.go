package metaengine

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// RowScanner is the minimal row surface shared by *sql.Rows and the
// dialect-test seams — the scan seam for the shared vector-scan core.
type RowScanner interface {
	Scan(dest ...any) error
}

// ScanScoredVector decodes one meta_vector row (id, raw vector, raw metadata),
// applies the metadata filters, and scores the survivor; ok=false when a
// filter excluded the row. decode turns the raw vector bytes into components —
// [DecodeVectorF32] for raw-LE-blob columns (mysql/sqlite/pg engines),
// [DecodeVectorJSON] for JSON-cast text columns (duckdb engine).
//
// This is the shared core of the SQL engines' VectorSearchFiltered scan loops
// (formerly three dialect twins; consolidated 2026-09-20).
func ScanScoredVector(
	rows RowScanner,
	query []float32,
	metric string,
	filters []VectorFilter,
	decode func([]byte) ([]float32, error),
) (VectorResult, bool, error) {
	var id string

	var vec, metaRaw []byte

	if err := rows.Scan(&id, &vec, &metaRaw); err != nil {
		return VectorResult{}, false, fmt.Errorf("scan %s: %w", id, err)
	}

	var meta map[string]any

	if metaRaw != nil {
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return VectorResult{}, false, fmt.Errorf("metadata %s: %w", id, err)
		}
	}

	if !VectorMatchesFilters(meta, filters) {
		return VectorResult{}, false, nil
	}

	values, err := decode(vec)
	if err != nil {
		return VectorResult{}, false, fmt.Errorf("decode %s: %w", id, err)
	}

	return VectorResult{
		ID:       id,
		Distance: VectorDistance(query, values, metric),
	}, true, nil
}

// ScanVectorResults drains a SQL-pushdown k-NN result set — (id, distance)
// rows in ascending distance order — into VectorResults, casting the scanned
// float64 distance to float32. The label is used as the error prefix
// (e.g. "duckdbengine.VectorSearch"); the rows.Err wrap intentionally carries
// no sub-prefix, matching the engines' original error strings.
func ScanVectorResults(rows *sql.Rows, label string) ([]VectorResult, error) {
	var results []VectorResult

	for rows.Next() {
		var id string

		var dist float64

		if err := rows.Scan(&id, &dist); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", label, err)
		}

		results = append(results, VectorResult{ID: id, Distance: float32(dist)})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}

	return results, nil
}
