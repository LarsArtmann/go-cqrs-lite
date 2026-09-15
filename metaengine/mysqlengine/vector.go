package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend + VectorFilterBackend + VectorCounter (degraded, brute-force) ---
//
// Embeddings live in meta_vector as bare little-endian float32 LONGBLOBs
// (metaengine.EncodeVectorF32); VectorSearch streams the collection's rows
// and computes every distance in Go — O(N·D) per query, declared as
// ComplexityON + degraded in the profile. Open-source MySQL and MariaDB
// expose no engine-side vector scoring usable across both dialects, so this
// path exists so single-engine deployments still serve vector queries
// (graceful degradation, never failure). MariaDB 11.7+ ships a native
// VECTOR column type + VEC_DISTANCE_* functions — engine-side pushdown is a
// tracked ROADMAP item gated on dialect/version detection.
//
// Distance and filter semantics are shared with every other brute-force
// engine via metaengine.VectorDistance / VectorMatchesFilters / TopKNearest,
// so adttest.RunMatrix parity holds against the memory engine's index.
//
// art-dupl:accept scan/filter bodies; sqliteengine vector.go is a dep-isolated dialect twin

const vectorInsertSQL = `INSERT INTO meta_vector (collection, id, vec, metadata)
	VALUES (?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		vec = VALUES(vec),
		metadata = VALUES(metadata)`

// VectorInsert adds an embedding to the collection. Upsert semantics: an
// existing (collection, id) row is fully replaced — upserting without
// metadata clears the old set.
func (e *mysqlEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	var metaJSON any // nil → SQL NULL
	if emb.Metadata != nil {
		data, err := json.Marshal(emb.Metadata)
		if err != nil {
			return fmt.Errorf("mysqlengine.VectorInsert: marshal metadata: %w", err)
		}

		metaJSON = string(data)
	}

	if _, err := e.conn().ExecContext(ctx, vectorInsertSQL,
		collection, emb.ID, metaengine.EncodeVectorF32(emb.Values), metaJSON,
	); err != nil {
		return fmt.Errorf("mysqlengine.VectorInsert: %w", err)
	}

	return nil
}

// VectorSearch returns the k nearest neighbors: scan + Go-side scoring.
func (e *mysqlEngine) VectorSearch(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	return e.vectorScan(ctx, collection, query, k, metric, nil)
}

// VectorSearchFiltered is the metadata-filtered k-NN path: filters apply
// BEFORE ranking, so the k results are the k nearest MATCHING neighbors.
// Filters are evaluated in Go so AND semantics match every other engine.
func (e *mysqlEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	return e.vectorScan(ctx, collection, query, k, metric, filters)
}

func (e *mysqlEngine) vectorScan(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	rows, err := e.conn().QueryContext(ctx,
		"SELECT id, vec, metadata FROM meta_vector WHERE collection = ?", collection)
	//art-dupl:accept dep-isolated dialect twin (duckdbengine/sqliteengine vector.go)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.vectorScan: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		res, ok, err := scanScoredVector(rows, query, metric, filters)
		if err != nil {
			return nil, fmt.Errorf("mysqlengine.vectorScan: %w", err)
		}

		if ok {
			results = append(results, res)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysqlengine.vectorScan: %w", err)
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
func (e *mysqlEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	var n int64

	err := e.conn().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM meta_vector WHERE collection = ?", collection).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("mysqlengine.VectorCount: %w", err)
	}

	return n, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *mysqlEngine) VectorCollections(ctx context.Context) ([]string, error) {
	rows, err := e.conn().QueryContext(ctx, "SELECT DISTINCT collection FROM meta_vector")
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.VectorCollections: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var collections []string

	for rows.Next() {
		var col string

		if err := rows.Scan(&col); err != nil {
			return nil, fmt.Errorf("mysqlengine.VectorCollections: scan: %w", err)
		}

		collections = append(collections, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysqlengine.VectorCollections: %w", err)
	}

	return collections, nil
}

var (
	_ metaengine.VectorBackend       = (*mysqlEngine)(nil)
	_ metaengine.VectorFilterBackend = (*mysqlEngine)(nil)
	_ metaengine.VectorCounter       = (*mysqlEngine)(nil)
)
