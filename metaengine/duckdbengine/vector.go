package duckdbengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend + VectorFilterBackend + VectorCounter (degraded, engine-scored) ---
//
// Embeddings live in meta_vector as FLOAT[] lists; k-NN is pushed fully into
// SQL — DuckDB core ships array_distance (L2), array_cosine_distance, and
// array_negative_inner_product, which map 1:1 onto metaengine.VectorDistance
// semantics (cosine = 1-cosSim, dot = negated, ascending = nearest). The
// engine evaluates every row (O(N), vectorized in C++) but has no ANN index:
// declared ComplexityON + degraded. The VSS extension adds HNSW indexes over
// the same ORDER BY distance LIMIT k shape — a tracked ROADMAP item.
//
// Parameters cross the wire as JSON text and cast in SQL ('[1.0,2.0]'::FLOAT[]
// and ::JSON), so no driver-specific array binding is needed. Filtered k-NN
// filters in Go (AND semantics shared with every engine) after fetching rows
// as JSON text.
//
// art-dupl:accept scan/filter bodies; sqliteengine vector.go is a dep-isolated dialect twin

const vectorInsertSQL = `INSERT INTO meta_vector (collection, id, vec, metadata)
	VALUES (?, ?, ?::FLOAT[], ?::JSON)
	ON CONFLICT (collection, id) DO UPDATE SET
		vec = excluded.vec,
		metadata = excluded.metadata`

// vectorTableDDL appends the meta_vector table to the engine's base schema.
// The PRIMARY KEY (collection, id) index also serves collection-prefix scans.
const vectorTableDDL = `
		CREATE TABLE IF NOT EXISTS meta_vector (
			collection VARCHAR NOT NULL,
			id VARCHAR NOT NULL,
			vec FLOAT[] NOT NULL,
			metadata JSON,
			PRIMARY KEY (collection, id)
		)`

// duckdbDistanceExpr maps a metric to a DuckDB SQL expression over the stored
// vec column. The core array_* functions only bind fixed-size ARRAY types,
// so both sides cast to FLOAT[dim] (dim = query length, an int formatted into
// the SQL text — never user data). All three already return distances in
// metaengine semantics (ascending = nearest-first).
func duckdbDistanceExpr(metric string, dim int) string {
	vecCast := fmt.Sprintf("CAST(vec AS FLOAT[%d])", dim)
	queryCast := fmt.Sprintf("?::FLOAT[%d]", dim)

	switch metric {
	case "cosine":
		return "array_cosine_distance(" + vecCast + ", " + queryCast + ")"
	case "dot":
		return "array_negative_inner_product(" + vecCast + ", " + queryCast + ")"
	default: // "euclidean", "", unknown → euclidean (matches computeDistance)
		return "array_distance(" + vecCast + ", " + queryCast + ")"
	}
}

// VectorInsert adds an embedding to the collection. Upsert semantics: an
// existing (collection, id) row is fully replaced — upserting without
// metadata clears the old set. Enforces the collection's dimension lock
// (first insert establishes the dimension; mismatching inserts are rejected
// with metaengine.ErrVectorDimensionMismatch).
func (e *duckdbEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	//art-dupl:accept dep-isolated dialect twin (pgengine/sqliteengine vector.go)
	var established int

	err := e.conn().QueryRowContext(ctx,
		"SELECT len(vec) FROM meta_vector WHERE collection = ? LIMIT 1", collection).
		Scan(&established)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("duckdbengine.VectorInsert: dimension probe: %w", err)
	}

	if err := metaengine.CheckVectorDimension(
		collection,
		established,
		len(emb.Values),
	); err != nil {
		return fmt.Errorf("duckdbengine.VectorInsert: %w", err)
	}

	vecJSON, err := json.Marshal(emb.Values)
	if err != nil {
		return fmt.Errorf("duckdbengine.VectorInsert: marshal: %w", err)
	}

	var metaJSON any // nil → SQL NULL
	if emb.Metadata != nil {
		data, err := json.Marshal(emb.Metadata)
		if err != nil {
			return fmt.Errorf("duckdbengine.VectorInsert: marshal metadata: %w", err)
		}

		metaJSON = string(data)
	}

	if _, err := e.conn().ExecContext(ctx, vectorInsertSQL,
		collection, emb.ID, string(vecJSON), metaJSON,
	); err != nil {
		return fmt.Errorf("duckdbengine.VectorInsert: %w", err)
	}

	return nil
}

// VectorSearch returns the k nearest neighbors via full SQL pushdown:
// distance, ordering, and limit all evaluate engine-side.
func (e *duckdbEngine) VectorSearch(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	if len(query) == 0 { // no dimension to cast to — score in Go
		return e.VectorSearchFiltered(ctx, collection, query, k, metric, nil)
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearch: marshal query: %w", err)
	}

	rows, err := e.conn().QueryContext(ctx,
		"SELECT id, "+duckdbDistanceExpr(metric, len(query))+" AS d FROM meta_vector "+
			"WHERE collection = ? ORDER BY d LIMIT ?",
		string(queryJSON), collection, k)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		var id string

		var dist float64

		if err := rows.Scan(&id, &dist); err != nil {
			return nil, fmt.Errorf("duckdbengine.VectorSearch: scan: %w", err)
		}

		results = append(results, metaengine.VectorResult{ID: id, Distance: float32(dist)})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearch: %w", err)
	}

	return results, nil
}

// VectorSearchFiltered is the metadata-filtered k-NN path: filters apply
// BEFORE ranking in Go (AND semantics shared with every engine), then the
// survivors are scored via metaengine.VectorDistance.
func (e *duckdbEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	rows, err := e.conn().QueryContext(ctx,
		"SELECT id, CAST(vec AS VARCHAR), CAST(metadata AS VARCHAR) "+
			"FROM meta_vector WHERE collection = ?", collection)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearchFiltered: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		res, ok, err := scanScoredVector(rows, query, metric, filters)
		if err != nil {
			return nil, fmt.Errorf("duckdbengine.VectorSearchFiltered: %w", err)
		}

		if ok {
			results = append(results, res)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearchFiltered: %w", err)
	}

	return metaengine.TopKNearest(results, k), nil
}

// scanScoredVector decodes one meta_vector row (vec/metadata fetched as JSON
// text via CAST), applies the metadata filters, and scores the survivor;
// ok=false when a filter excluded the row.
func scanScoredVector(
	rows rowScanner,
	query []float32,
	metric string,
	filters []metaengine.VectorFilter,
) (res metaengine.VectorResult, ok bool, err error) {
	var id string

	var vec, metaRaw *string

	if err := rows.Scan(&id, &vec, &metaRaw); err != nil {
		return metaengine.VectorResult{}, false, fmt.Errorf("scan %s: %w", id, err)
	}

	var meta map[string]any
	if metaRaw != nil {
		if err := json.Unmarshal([]byte(*metaRaw), &meta); err != nil {
			return metaengine.VectorResult{}, false, fmt.Errorf("metadata %s: %w", id, err)
		}
	}

	if !metaengine.VectorMatchesFilters(meta, filters) {
		return metaengine.VectorResult{}, false, nil
	}

	values, err := metaengine.DecodeVectorJSON([]byte(*vec))
	if err != nil {
		return metaengine.VectorResult{}, false, fmt.Errorf("decode %s: %w", id, err)
	}

	return metaengine.VectorResult{
		ID:       id,
		Distance: metaengine.VectorDistance(query, values, metric),
	}, true, nil
}

// rowScanner is the Scan surface of *sql.Rows (interface seam keeps the
// dialect-twin bodies comparable).
type rowScanner interface {
	Scan(dest ...any) error
}

// VectorCount returns the number of embeddings in the collection via SQL
// COUNT — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *duckdbEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	//art-dupl:accept dep-isolated dialect twin (mysqlengine/pgengine VectorCount)
	var n int64

	err := e.conn().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM meta_vector WHERE collection = ?", collection).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("duckdbengine.VectorCount: %w", err)
	}

	return n, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *duckdbEngine) VectorCollections(ctx context.Context) ([]string, error) {
	//art-dupl:accept dep-isolated dialect twin (mysqlengine VectorCollections)
	rows, err := e.conn().QueryContext(ctx, "SELECT DISTINCT collection FROM meta_vector")
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorCollections: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var collections []string

	for rows.Next() {
		var col string

		if err := rows.Scan(&col); err != nil {
			return nil, fmt.Errorf("duckdbengine.VectorCollections: scan: %w", err)
		}

		collections = append(collections, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorCollections: %w", err)
	}

	return collections, nil
}

var (
	_ metaengine.VectorBackend       = (*duckdbEngine)(nil)
	_ metaengine.VectorFilterBackend = (*duckdbEngine)(nil)
	_ metaengine.VectorCounter       = (*duckdbEngine)(nil)
)

// VectorSearchPath reports the engine-side SQL scoring path (implements
// [metaengine.VectorPathReporter]). Edge case: an empty query vector has no
// dimension to cast to, so that one query shape degrades to a Go-scored
// scan; the label reflects the normal (dimensioned) path.
func (e *duckdbEngine) VectorSearchPath() string {
	return metaengine.VectorPathPushdown
}

var _ metaengine.VectorPathReporter = (*duckdbEngine)(nil)
