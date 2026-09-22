package metaengine

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
)

// ScanVectorDimensionProbe scans the result of a dialect-specific dimension
// probe query ("SELECT len(vec)|LENGTH(vec)/4|jsonb_array_length(...) ...
// LIMIT 1") into the collection's established dimension. sql.ErrNoRows passes
// as 0 — an empty collection has no lock yet; every other error wraps as
// "<label>: dimension probe". It is the shared probe step of the SQL engines'
// VectorInsert (the query text stays engine-local because each dialect
// measures vector length differently).
func ScanVectorDimensionProbe(row *sql.Row, label string) (int, error) {
	var established int

	if err := row.Scan(&established); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%s: dimension probe: %w", label, err)
	}

	return established, nil
}

// VectorMetadataArg converts an embedding's metadata into the SQL argument
// for the metadata column: nil metadata → nil (binds SQL NULL); otherwise the
// JSON encoding as a string (json/v2 semantics, matching the engines'
// original marshals byte for byte). The label is used as the error prefix.
func VectorMetadataArg(emb Embedding, label string) (any, error) {
	if emb.Metadata == nil {
		return nil, nil
	}

	data, err := json.Marshal(emb.Metadata)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal metadata: %w", label, err)
	}

	return string(data), nil
}
