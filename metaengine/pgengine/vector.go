package pgengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend + VectorFilterBackend (degraded, brute-force) ---
//
// Embeddings are stored in meta_vector as JSONB. VectorSearch streams the
// collection's rows and computes every distance in Go — O(N·D) per query,
// declared as ComplexityON + degraded in the profile. Postgres without
// pgvector has no ANN index; this path exists so single-engine deployments
// still serve vector queries (graceful degradation, never failure). For
// production scale, deploy pgvector or another ANN-capable engine and let
// the planner route ADTVector there.
//
// Distance and filter semantics are shared with every other brute-force
// engine via metaengine.VectorDistance / VectorMatchesFilters / TopKNearest,
// so adttest.RunMatrix parity holds against the memory engine's index.

func (e *pgEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	vecJSON, err := json.Marshal(emb.Values)
	if err != nil {
		return fmt.Errorf("pgengine.VectorInsert: marshal: %w", err)
	}

	var metaJSON any // nil marshals to SQL NULL
	if emb.Metadata != nil {
		metaJSON = string(mustMarshalMetadata(emb.Metadata))
	}

	_, err = e.conn().ExecContext(
		ctx,
		`INSERT INTO meta_vector (collection, id, vector, metadata)
		 VALUES ($1, $2, $3::jsonb, $4::jsonb)
		 ON CONFLICT (collection, id) DO UPDATE SET
			vector = excluded.vector,
			metadata = excluded.metadata`,
		collection, emb.ID, string(vecJSON), metaJSON,
	)
	if err != nil {
		return fmt.Errorf("pgengine.VectorInsert: %w", err)
	}

	return nil
}

func (e *pgEngine) VectorSearch(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT id, vector::text FROM meta_vector WHERE collection = $1`,
		collection,
	)
	// art-dupl:accept scan prologue; pebbleengine vector.go is a dep-isolated dialect twin
	if err != nil {
		return nil, fmt.Errorf("pgengine.VectorSearch: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		var id string

		var raw []byte

		if err := rows.Scan(&id, &raw); err != nil {
			return nil, fmt.Errorf("pgengine.VectorSearch: scan: %w", err)
		}

		// JSONB column keeps this scan JSON-decoding on purpose: the binary
		// float32 payload format (metaengine.EncodeVectorBinary) needs a raw
		// BYTEA column, which is a DDL migration on existing deployments —
		// deferred until the KV-engine win proves insufficient here.
		vec, err := metaengine.DecodeVectorJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("pgengine.VectorSearch: decode %s: %w", id, err)
		}

		results = append(results, metaengine.VectorResult{
			ID:       id,
			Distance: metaengine.VectorDistance(query, vec, metric),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgengine.VectorSearch: %w", err)
	}

	return metaengine.TopKNearest(results, k), nil
}

// VectorSearchFiltered is the metadata-filtered k-NN path: it scans the
// collection, applies the metadata filters, and only then scores survivors.
// Embeddings excluded by a filter never pay the distance computation.
func (e *pgEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT id, vector::text, metadata::text FROM meta_vector WHERE collection = $1`,
		collection,
	)
	// art-dupl:accept scan prologue; pebbleengine vector.go is a dep-isolated dialect twin
	if err != nil {
		return nil, fmt.Errorf("pgengine.VectorSearchFiltered: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var results []metaengine.VectorResult

	for rows.Next() {
		var id string

		var raw, metaRaw []byte

		if err := rows.Scan(&id, &raw, &metaRaw); err != nil {
			return nil, fmt.Errorf("pgengine.VectorSearchFiltered: scan: %w", err)
		}

		var meta map[string]any
		if metaRaw != nil {
			if err := json.Unmarshal(metaRaw, &meta); err != nil {
				return nil, fmt.Errorf("pgengine.VectorSearchFiltered: metadata %s: %w", id, err)
			}
		}

		if !metaengine.VectorMatchesFilters(meta, filters) {
			continue
		}

		vec, err := metaengine.DecodeVectorJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("pgengine.VectorSearchFiltered: decode %s: %w", id, err)
		}

		results = append(results, metaengine.VectorResult{
			ID:       id,
			Distance: metaengine.VectorDistance(query, vec, metric),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgengine.VectorSearchFiltered: %w", err)
	}

	return metaengine.TopKNearest(results, k), nil
}

// mustMarshalMetadata encodes filter metadata; map[string]any from the typed
// Embedding field always marshals, so a failure is a programming error.
func mustMarshalMetadata(meta map[string]any) []byte {
	data, err := json.Marshal(meta)
	if err != nil {
		panic(fmt.Sprintf("pgengine: metadata marshal: %v", err))
	}

	return data
}

// VectorCount returns the number of embeddings in the collection via SQL
// COUNT — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *pgEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	var n int64

	err := e.conn().QueryRowContext(
		ctx,
		`SELECT count(*) FROM meta_vector WHERE collection = $1`,
		collection,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("pgengine.VectorCount: %w", err)
	}

	return n, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *pgEngine) VectorCollections(ctx context.Context) ([]string, error) {
	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT DISTINCT collection FROM meta_vector ORDER BY collection`,
	)
	if err != nil {
		return nil, fmt.Errorf("pgengine.VectorCollections: %w", err)
	}
	defer metaengine.DeferClose(rows)

	var out []string

	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, fmt.Errorf("pgengine.VectorCollections: scan: %w", err)
		}

		out = append(out, col)
	}

	return out, rows.Err()
}
