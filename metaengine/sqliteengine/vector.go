package sqliteengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend + VectorFilterBackend + VectorCounter (degraded, brute-force) ---
//
// Embeddings live in meta_vector as bare little-endian float32 BLOBs
// (metaengine.EncodeVectorF32 — byte-compatible with libSQL's F32_BLOB, so
// the same rows score in SQL or in Go). Two execution paths, chosen by a
// construction-time driver probe:
//
//   - modernc pure-Go SQLite: scan rows and score in Go — O(N·D) per query.
//   - libSQL drivers (tursoengine delegates here): k-NN pushed into SQL via
//     vector32()/vector_distance_* — same O(N) complexity, engine-side scoring.
//
// Distance semantics are metaengine.VectorDistance everywhere: cosine =
// 1-cosSim, dot = negated (ascending = nearest). This path exists so
// single-engine deployments still serve vector queries (graceful
// degradation, never failure); for production scale deploy an ANN-capable
// engine and let the planner route ADTVector there.
//
// art-dupl:accept scan/filter bodies; mysqlengine vector.go is a dep-isolated dialect twin

const vectorInsertSQL = `INSERT INTO meta_vector (collection, id, vec, metadata)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(collection, id) DO UPDATE SET
		vec = excluded.vec,
		metadata = excluded.metadata`

// libSQLProbeSQL detects libSQL vector functions at construction: modernc
// fails here ("no such function"), turso (embedded or remote libSQL) succeeds.
const libSQLProbeSQL = `SELECT vector_distance_cos(vector32('[1]'), vector32('[1]'))`

func probeVectorSQL(db *sql.DB) bool {
	var got float64

	return db.QueryRowContext(context.Background(), libSQLProbeSQL).Scan(&got) == nil
}

// libSQLDistanceExpr maps a metric to a libSQL SQL expression over the stored
// vec column and a vector32(?) placeholder, preserving VectorDistance
// semantics (dot is negated so ascending order is nearest-first).
func libSQLDistanceExpr(metric string) string {
	switch metric {
	case "cosine":
		return "vector_distance_cos(vec, vector32(?))"
	case "dot":
		return "(-vector_distance_dot(vec, vector32(?)))"
	default: // "euclidean", "", unknown → euclidean (matches computeDistance)
		return "vector_distance_l2(vec, vector32(?))"
	}
}

// VectorInsert adds an embedding to the collection. Upsert semantics: an
// existing (collection, id) row is fully replaced — upserting without
// metadata clears the old set.
func (e *sqliteEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	var metaJSON any // nil marshals to SQL NULL
	if emb.Metadata != nil {
		data, err := json.Marshal(emb.Metadata)
		if err != nil {
			return fmt.Errorf("sqliteengine.VectorInsert: marshal metadata: %w", err)
		}

		metaJSON = string(data)
	}

	if _, err := e.xc().exec(
		ctx, vectorInsertSQL, collection, emb.ID, metaengine.EncodeVectorF32(emb.Values), metaJSON,
	); err != nil {
		return fmt.Errorf("sqliteengine.VectorInsert: %w", err)
	}

	return nil
}

// VectorSearch returns the k nearest neighbors. libSQL drivers score in SQL;
// modernc scans and scores in Go.
func (e *sqliteEngine) VectorSearch(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	if e.vectorSQL {
		return e.vectorSearchPushdown(ctx, collection, query, k, metric)
	}

	return e.vectorScan(ctx, collection, query, k, metric, nil)
}

func (e *sqliteEngine) vectorSearchPushdown(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.VectorSearch: marshal query: %w", err)
	}

	rows, err := e.xc().query(ctx,
		"SELECT id, "+libSQLDistanceExpr(metric)+" AS d FROM meta_vector "+
			"WHERE collection = ? ORDER BY d LIMIT ?",
		string(queryJSON), collection, k)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		var id string

		var dist float64

		if err := rows.Scan(&id, &dist); err != nil {
			return nil, fmt.Errorf("sqliteengine.VectorSearch: scan: %w", err)
		}

		results = append(results, metaengine.VectorResult{ID: id, Distance: float32(dist)})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqliteengine.VectorSearch: %w", err)
	}

	return results, nil
}

// VectorSearchFiltered is the metadata-filtered k-NN path: filters apply
// BEFORE ranking, so the k results are the k nearest MATCHING neighbors.
// Filters are evaluated in Go so AND semantics match every other engine.
func (e *sqliteEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	return e.vectorScan(ctx, collection, query, k, metric, filters)
}

func (e *sqliteEngine) vectorScan(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	rows, err := e.xc().query(ctx,
		"SELECT id, vec, metadata FROM meta_vector WHERE collection = ?", collection)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.vectorScan: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		res, ok, err := scanScoredVector(rows, query, metric, filters)
		if err != nil {
			return nil, fmt.Errorf("sqliteengine.vectorScan: %w", err)
		}

		if ok {
			results = append(results, res)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqliteengine.vectorScan: %w", err)
	}

	return metaengine.TopKNearest(results, k), nil
}

// scanScoredVector decodes one meta_vector row, applies the metadata filters,
// and scores the survivor; ok=false when a filter excluded the row.
func scanScoredVector(
	rows *sql.Rows,
	query []float32,
	metric string,
	filters []metaengine.VectorFilter,
) (res metaengine.VectorResult, ok bool, err error) {
	var id string

	var vec, metaRaw []byte

	if err := rows.Scan(&id, &vec, &metaRaw); err != nil {
		return metaengine.VectorResult{}, false, fmt.Errorf("scan %s: %w", id, err)
	}

	var meta map[string]any
	if metaRaw != nil {
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return metaengine.VectorResult{}, false, fmt.Errorf("metadata %s: %w", id, err)
		}
	}

	if !metaengine.VectorMatchesFilters(meta, filters) {
		return metaengine.VectorResult{}, false, nil
	}

	values, err := metaengine.DecodeVectorF32(vec)
	if err != nil {
		return metaengine.VectorResult{}, false, fmt.Errorf("decode %s: %w", id, err)
	}

	return metaengine.VectorResult{
		ID:       id,
		Distance: metaengine.VectorDistance(query, values, metric),
	}, true, nil
}

// VectorCount returns the number of embeddings in the collection via SQL
// COUNT — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *sqliteEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	var n int64

	err := e.xc().queryRow(ctx,
		"SELECT COUNT(*) FROM meta_vector WHERE collection = ?", collection).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("sqliteengine.VectorCount: %w", err)
	}

	return n, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *sqliteEngine) VectorCollections(ctx context.Context) ([]string, error) {
	rows, err := e.xc().query(ctx, "SELECT DISTINCT collection FROM meta_vector")
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.VectorCollections: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var collections []string

	for rows.Next() {
		var col string

		if err := rows.Scan(&col); err != nil {
			return nil, fmt.Errorf("sqliteengine.VectorCollections: scan: %w", err)
		}

		collections = append(collections, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqliteengine.VectorCollections: %w", err)
	}

	return collections, nil
}

var (
	_ metaengine.VectorBackend       = (*sqliteEngine)(nil)
	_ metaengine.VectorFilterBackend = (*sqliteEngine)(nil)
	_ metaengine.VectorCounter       = (*sqliteEngine)(nil)
)
