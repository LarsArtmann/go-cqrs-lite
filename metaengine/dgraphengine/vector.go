package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- VectorBackend + VectorFilterBackend + VectorCounter (degraded, brute-force) ---
//
// One node per (collection, id) pair mirrors MapSet's upsert shape. Values
// live in the cqrs.vector_values predicate typed float32vector (Dgraph v24+),
// mutated as a quoted JSON array string — Dgraph rejects bare JSON arrays for
// vector predicates. VectorSearch streams the collection's nodes over gRPC
// and scores in Go — O(N·D) per query, declared ComplexityON + degraded.
// Dgraph's native ANN (hnsw index + similar_to) is metric-coupled at schema
// time and returns uids without distances, so it stays a tracked ROADMAP
// item; this path exists so single-engine deployments still serve vector
// queries (graceful degradation, never failure).
//
// Distance and filter semantics are shared with every other brute-force
// engine via metaengine.VectorDistance / VectorMatchesFilters / TopKNearest,
// so adttest.RunMatrix parity holds against the memory engine's index.

const vectorEntryQuery = `query entry($col: string, $id: string) {
	entry as var(func: eq(cqrs.vector_collection, $col)) @filter(eq(cqrs.vector_id, $id))
}`

// formatVectorValues renders values as the quoted JSON string Dgraph's
// float32vector predicates accept ("[1.0, 2.0]").
func formatVectorValues(values []float32) (string, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal values: %w", err)
	}

	return string(data), nil
}

// VectorInsert adds an embedding to the collection. Upsert semantics: an
// existing (collection, id) node is fully replaced — upserting without
// metadata clears the old set.
func (e *dgraphEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	if err := e.ensureVectorSchema(ctx); err != nil {
		return err //nolint:wrapcheck // already actionable
	}

	established, err := e.establishedVectorDimension(ctx, collection)
	if err != nil {
		return err //nolint:wrapcheck // wrapped by the probe helper
	}

	if err := metaengine.CheckVectorDimension(collection, established, len(emb.Values)); err != nil {
		return fmt.Errorf("dgraphengine.VectorInsert: %w", err)
	}

	valuesStr, err := formatVectorValues(emb.Values)
	if err != nil {
		return fmt.Errorf("dgraphengine.VectorInsert: %w", err)
	}

	create := map[string]any{
		"uid":                    "_:new",
		"cqrs.vector_collection": collection,
		"cqrs.vector_id":         emb.ID,
		"cqrs.vector_values":     valuesStr,
		"dgraph.type":            []string{"VectorEmbedding"},
	}

	update := map[string]any{
		"uid":                "uid(entry)",
		"cqrs.vector_values": valuesStr,
	}

	if emb.Metadata != nil {
		metaStr, err := json.Marshal(emb.Metadata)
		if err != nil {
			return fmt.Errorf("dgraphengine.VectorInsert: marshal metadata: %w", err)
		}

		create["cqrs.vector_metadata"] = string(metaStr)
		update["cqrs.vector_metadata"] = string(metaStr)
	}

	req, err := vectorUpsertRequest(collection, emb.ID, create, update,
		emb.Metadata == nil)
	if err != nil {
		return fmt.Errorf("dgraphengine.VectorInsert: %w", err)
	}

	if _, err := e.doWrite(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.VectorInsert: %w", err)
	}

	return nil
}

// vectorSchema is the LAZY vector predicate schema. It is deliberately NOT
// in init(): `float32vector` is rejected by Dgraph < v24, and a
// construction-time Alter would break engine startup for every pre-v24
// deployment — including ones that never use vectors. First vector use
// applies the schema; old servers then fail at first vector use with an
// actionable error instead of failing to boot (README documents the v24+
// floor for vector ops).
const vectorSchema = `
		cqrs.vector_collection: string @index(exact) @upsert .
		cqrs.vector_id: string @index(exact) @upsert .
		cqrs.vector_values: float32vector .
		cqrs.vector_metadata: string .
	`

// ensureVectorSchema lazily applies the vector predicate schema on first
// vector use (appliedSchemas makes the steady state one map lookup).
func (e *dgraphEngine) ensureVectorSchema(ctx context.Context) error {
	e.schemaMu.Lock()
	defer e.schemaMu.Unlock()

	if e.appliedSchemas["vector"] {
		return nil
	}

	if err := e.retryOnContention(ctx, false, func() error {
		return e.client.Alter(ctx, &api.Operation{Schema: vectorSchema})
	}); err != nil {
		return fmt.Errorf("dgraphengine: vector predicates require Dgraph v24+ "+
			"(float32vector type rejected): %w", err)
	}

	e.appliedSchemas["vector"] = true

	return nil
}

// vectorDimensionQuery reads the collection's first vector for the
// insert-time dimension lock (metaengine.CheckVectorDimension).
const vectorDimensionQuery = `query dims($col: string) {
	dims(func: eq(cqrs.vector_collection, $col), first: 1) {
		cqrs.vector_values
	}
}`

// establishedVectorDimension returns the stored dimension of the
// collection's first vector (0 when the collection is empty).
func (e *dgraphEngine) establishedVectorDimension(
	ctx context.Context,
	collection string,
) (int, error) {
	var out vectorRows

	if err := vectorQuery(e, ctx, vectorDimensionQuery,
		map[string]string{"$col": collection}, &out); err != nil {
		return 0, fmt.Errorf("dgraphengine.VectorInsert: dimension probe: %w", err)
	}

	if len(out.Vecs) == 0 {
		return 0, nil
	}

	return len(out.Vecs[0].Values), nil
}

// vectorUpsertRequest assembles the conditional upsert. clearMetadata adds a
// delete mutation removing stale cqrs.vector_metadata on update-without-
// metadata (setting "" would leave a set predicate, not an absent one).
func vectorUpsertRequest(
	collection, id string,
	create, update map[string]any,
	clearMetadata bool,
) (*api.Request, error) {
	createJSON, err := json.Marshal(create)
	if err != nil {
		return nil, fmt.Errorf("marshal create: %w", err)
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("marshal update: %w", err)
	}

	req := &api.Request{
		Query: vectorEntryQuery,
		Vars:  map[string]string{"$col": collection, "$id": id},
		Mutations: []*api.Mutation{
			{SetJson: createJSON, Cond: "@if(eq(len(entry), 0))"},
			{SetJson: updateJSON, Cond: "@if(eq(len(entry), 1))"},
		},
	}

	if clearMetadata {
		deleteJSON, _ := json.Marshal(map[string]any{
			"uid":                  "uid(entry)",
			"cqrs.vector_metadata": nil,
		})
		req.Mutations = append(req.Mutations,
			&api.Mutation{DeleteJson: deleteJSON, Cond: "@if(eq(len(entry), 1))"})
	}

	return req, nil
}

// vectorRows is the decoded shape of the collection scan.
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

var (
	_ metaengine.VectorBackend       = (*dgraphEngine)(nil)
	_ metaengine.VectorFilterBackend = (*dgraphEngine)(nil)
	_ metaengine.VectorCounter       = (*dgraphEngine)(nil)
)

// VectorSearchPath reports the Go-scored scan path (implements
// [metaengine.VectorPathReporter]): this engine has no native vector
// distance function, so k-NN scans rows and scores via
// metaengine.VectorDistance (ADR-0140).
func (e *dgraphEngine) VectorSearchPath() string {
	return metaengine.VectorPathScan
}

var (
	_ metaengine.VectorPathReporter = (*dgraphEngine)(nil)
)
