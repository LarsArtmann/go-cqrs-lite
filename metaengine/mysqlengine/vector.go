package mysqlengine

import (
	"context"
	"database/sql"
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
// metadata clears the old set. Enforces the collection's dimension lock
// (first insert establishes the dimension; mismatching inserts are rejected
// with metaengine.ErrVectorDimensionMismatch).
func (e *mysqlEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	established, err := metaengine.ScanVectorDimensionProbe(
		e.conn(ctx).QueryRowContext(ctx,
			// CAST ... AS SIGNED: MySQL/MariaDB "/" is DECIMAL division ("2.0000"
			// scans as []uint8, not an int); both dialects cast to integer here.
			"SELECT CAST(LENGTH(vec)/4 AS SIGNED) FROM meta_vector WHERE collection = ? LIMIT 1",
			collection),
		"mysqlengine.VectorInsert")
	if err != nil {
		return err
	}

	if err := metaengine.CheckVectorDimension(
		collection,
		established,
		len(emb.Values),
	); err != nil {
		return fmt.Errorf("mysqlengine.VectorInsert: %w", err)
	}

	metaJSON, err := metaengine.VectorMetadataArg(emb, "mysqlengine.VectorInsert")
	if err != nil {
		return err
	}

	if _, err := e.conn(ctx).ExecContext(ctx, vectorInsertSQL,
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
	rows, err := e.conn(ctx).QueryContext(ctx,
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

// scanScoredVector delegates to the shared metaengine.ScanScoredVector core
// (raw F32 vector blob, byte metadata). Consolidated from the dialect twin.
func scanScoredVector(
	rows *sql.Rows,
	query []float32,
	metric string,
	filters []metaengine.VectorFilter,
) (metaengine.VectorResult, bool, error) {
	return metaengine.ScanScoredVector(rows, query, metric, filters, metaengine.DecodeVectorF32)
}

// VectorCount returns the number of embeddings in the collection via SQL
// COUNT — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *mysqlEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	var n int64

	err := e.conn(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM meta_vector WHERE collection = ?", collection).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("mysqlengine.VectorCount: %w", err)
	}

	return n, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *mysqlEngine) VectorCollections(ctx context.Context) ([]string, error) {
	rows, err := e.conn(ctx).QueryContext(ctx, "SELECT DISTINCT collection FROM meta_vector")
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

// VectorSearchPath reports the Go-scored scan path (implements
// [metaengine.VectorPathReporter]): this engine has no native vector
// distance function, so k-NN scans rows and scores via
// metaengine.VectorDistance (ADR-0140).
func (e *mysqlEngine) VectorSearchPath() string {
	return metaengine.VectorPathScan
}

var _ metaengine.VectorPathReporter = (*mysqlEngine)(nil)
