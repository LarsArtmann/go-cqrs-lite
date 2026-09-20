package metaengine

import (
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
