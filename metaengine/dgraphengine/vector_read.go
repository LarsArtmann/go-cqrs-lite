package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Vector read side: k-NN scan + Go-side scoring, count, and collection
// enumeration. This engine has no native vector distance function, so k-NN
// scans rows and scores via metaengine.VectorDistance (ADR-0140) — graceful
// degradation, never failure; see the write-side header in vector.go.

type vectorRows struct {
	Vecs []vectorRow `json:"vecs"`
}

type vectorRow struct {
	ID       string    `json:"cqrs.vector_id"`
	Values   []float32 `json:"cqrs.vector_values"`
	Metadata string    `json:"cqrs.vector_metadata"`
}

const vectorScanQuery = `query vecs($col: string) {
	vecs(func: eq(cqrs.vector_collection, $col)) {
		cqrs.vector_id
		cqrs.vector_values
		cqrs.vector_metadata
	}
}`

// VectorSearch returns the k nearest neighbors: scan + Go-side scoring.
func (e *dgraphEngine) VectorSearch(
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
func (e *dgraphEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	return e.vectorScan(ctx, collection, query, k, metric, filters)
}

func (e *dgraphEngine) vectorScan(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	if err := e.ensureVectorSchema(ctx); err != nil {
		return nil, err //nolint:wrapcheck // already actionable
	}

	var out vectorRows

	if err := vectorQuery(e, ctx, vectorScanQuery,
		map[string]string{"$col": collection}, &out); err != nil {
		return nil, fmt.Errorf("dgraphengine.vectorScan: %w", err)
	}

	var results []metaengine.VectorResult

	for _, row := range out.Vecs {
		var meta map[string]any
		if row.Metadata != "" {
			if err := json.Unmarshal([]byte(row.Metadata), &meta); err != nil {
				return nil, fmt.Errorf("dgraphengine.vectorScan: metadata %s: %w", row.ID, err)
			}
		}

		if !metaengine.VectorMatchesFilters(meta, filters) {
			continue
		}

		results = append(results, metaengine.VectorResult{
			ID:       row.ID,
			Distance: metaengine.VectorDistance(query, row.Values, metric),
		})
	}

	return metaengine.TopKNearest(results, k), nil
}

// VectorCount returns the number of embeddings in the collection via a DQL
// count — no payload transfer. Implements the count member of
// [metaengine.VectorCounter].
func (e *dgraphEngine) VectorCount(ctx context.Context, collection string) (int64, error) {
	if err := e.ensureVectorSchema(ctx); err != nil {
		return 0, err //nolint:wrapcheck // already actionable
	}

	var out struct {
		Vecs []struct {
			Count int64 `json:"count"`
		} `json:"vecs"`
	}

	if err := vectorQuery(e, ctx,
		`query vecs($col: string) {
			vecs(func: eq(cqrs.vector_collection, $col)) { count(uid) }
		}`,
		map[string]string{"$col": collection}, &out); err != nil {
		return 0, fmt.Errorf("dgraphengine.VectorCount: %w", err)
	}

	if len(out.Vecs) == 0 {
		return 0, nil
	}

	return out.Vecs[0].Count, nil
}

// VectorCollections lists the collections holding at least one embedding.
// Implements the enumeration member of [metaengine.VectorCounter].
func (e *dgraphEngine) VectorCollections(ctx context.Context) ([]string, error) {
	if err := e.ensureVectorSchema(ctx); err != nil {
		return nil, err //nolint:wrapcheck // already actionable
	}

	var out struct {
		All []struct {
			Collection string `json:"cqrs.vector_collection"`
		} `json:"all"`
	}

	if err := vectorQuery(e, ctx,
		`{ all(func: has(cqrs.vector_collection)) { cqrs.vector_collection } }`,
		nil, &out); err != nil {
		return nil, fmt.Errorf("dgraphengine.VectorCollections: %w", err)
	}

	seen := make(map[string]bool, len(out.All))

	var collections []string

	for _, row := range out.All {
		if !seen[row.Collection] {
			seen[row.Collection] = true
			collections = append(collections, row.Collection)
		}
	}

	return collections, nil
}

// vectorQuoteHint: user data only ever crosses as $vars (QueryTo) or
// SetJson payloads — never formatted into DQL text.

// vectorQuery runs a DQL read through the active read transaction and
// decodes the JSON envelope into out.
func vectorQuery(
	e *dgraphEngine,
	ctx context.Context,
	q string,
	vars map[string]string,
	out any,
) error {
	resp, err := e.readTx().QueryWithVars(ctx, q, vars)
	if err != nil {
		return err //nolint:wrapcheck // caller wraps with the operation name
	}

	if err := json.Unmarshal(resp.GetJson(), out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// VectorSearchPath reports the Go-scored scan path (implements
// [metaengine.VectorPathReporter]): this engine has no native vector
// distance function, so k-NN scans rows and scores via
// metaengine.VectorDistance (ADR-0140).
func (e *dgraphEngine) VectorSearchPath() string {
	return metaengine.VectorPathScan
}

var _ metaengine.VectorPathReporter = (*dgraphEngine)(nil)
