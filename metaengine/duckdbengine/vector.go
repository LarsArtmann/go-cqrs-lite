package duckdbengine

import (
	"context"
	"encoding/json/v2"
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
	//art-dupl:accept dimension-lock idiom twin of pgengine VectorInsert; probe SQL, value marshaling, and exec surface are dialect-specific, adttest pins the semantics
	established, err := metaengine.ScanVectorDimensionProbe(
		e.conn(ctx).QueryRowContext(ctx,
			"SELECT len(vec) FROM meta_vector WHERE collection = ? LIMIT 1", collection),
		"duckdbengine.VectorInsert")
	if err != nil {
		return err
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

	metaJSON, err := metaengine.VectorMetadataArg(emb, "duckdbengine.VectorInsert")
	if err != nil {
		return err
	}

	if _, err := e.conn(ctx).ExecContext(ctx, vectorInsertSQL,
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

	rows, err := e.conn(ctx).QueryContext(ctx,
		"SELECT id, "+duckdbDistanceExpr(metric, len(query))+" AS d FROM meta_vector "+
			"WHERE collection = ? ORDER BY d LIMIT ?",
		string(queryJSON), collection, k)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(rows)

	return metaengine.ScanVectorResults(rows, "duckdbengine.VectorSearch")
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
	rows, err := e.conn(ctx).QueryContext(ctx,
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

// scanScoredVector delegates to the shared metaengine.ScanScoredVector core;
// the decode seam is DecodeVectorJSON because duckdb stores the vector as a
// JSON-cast text column (the other SQL engines use raw F32 blobs).
func scanScoredVector(
	rows metaengine.RowScanner,
	query []float32,
	metric string,
	filters []metaengine.VectorFilter,
) (metaengine.VectorResult, bool, error) {
	return metaengine.ScanScoredVector(rows, query, metric, filters, metaengine.DecodeVectorJSON)
}

// VectorCount returns the number of embeddings in the collection via SQL
// COUNT — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *duckdbEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	//art-dupl:accept dep-isolated dialect twin (mysqlengine/pgengine VectorCount)
	var n int64

	err := e.conn(ctx).QueryRowContext(ctx,
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
	rows, err := e.conn(ctx).QueryContext(ctx, "SELECT DISTINCT collection FROM meta_vector")
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
