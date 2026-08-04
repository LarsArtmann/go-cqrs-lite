package metaengine

import (
	"context"
	"reflect"
)

// extractVectorQuery reads the query vector, metric, and k from a query input
// struct by field name. Recognized field names:
//   - Values or Vector or Query ([]float32) — the query embedding
//   - Metric or Distance (string) — distance metric: "cosine", "euclidean", "dot"
//   - K or TopK or Limit (int) — number of neighbors to return
func extractVectorQuery(input any) (vec []float32, metric string, k int) {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, "euclidean", 10
	}

	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		name := field.Name

		switch {
		case (name == "Values" || name == "Vector" || name == "Query") && field.Type.Kind() == reflect.Slice:
			if s, ok := v.Field(i).Interface().([]float32); ok {
				vec = s
			}

		case name == "Metric" || name == "Distance":
			if s, ok := v.Field(i).Interface().(string); ok {
				metric = s
			}

		case (name == "K" || name == "TopK" || name == "Limit") && field.Type.Kind() == reflect.Int:
			k = int(v.Field(i).Int())
		}
	}

	if metric == "" {
		metric = "euclidean"
	}

	if k <= 0 {
		k = 10
	}

	return vec, metric, k
}

// extractSearchQuery reads the query text and limit from a query input struct.
// Recognized field names:
//   - Query or Text or Q (string) — the full-text query string
//   - Limit (int) — maximum results to return (default 10)
func extractSearchQuery(input any) (text string, limit int) {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", 10
	}

	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		name := field.Name

		switch {
		case (name == "Query" || name == "Text" || name == "Q") && field.Type.Kind() == reflect.String:
			text = v.Field(i).String()

		case name == "Limit" && field.Type.Kind() == reflect.Int:
			limit = int(v.Field(i).Int())
		}
	}

	if limit <= 0 {
		limit = 10
	}

	return text, limit
}

// executeSliceResult is the shared body of the Vector/Search/Spatial
// ExecuteTyped wrappers: execute, nil-check, type-assert to []R.
func executeSliceResult[R any](
	ctx context.Context,
	store *Store,
	input any,
) ([]R, error) {
	raw, err := store.ExecuteCtx(ctx, input)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, ErrNotFound
	}

	results, ok := raw.([]R)
	if !ok {
		return nil, errExecuteTypeMismatch
	}

	return results, nil
}

// VectorExecuteTyped is the type-safe wrapper for vector similarity search.
// It dispatches the query input and type-asserts the result to []VectorResult.
//
// Usage:
//
//	results, err := metaengine.VectorExecuteTyped[SemanticSearch](
//	    ctx, store, SemanticSearch{Vector: queryEmb, K: 5, Metric: "cosine"})
func VectorExecuteTyped[Q any](
	ctx context.Context,
	store *Store,
	input Q,
) ([]VectorResult, error) {
	return executeSliceResult[VectorResult](ctx, store, input)
}

// SearchExecuteTyped is the type-safe wrapper for full-text search.
// It dispatches the query input and type-asserts the result to []SearchResult.
//
// Usage:
//
//	results, err := metaengine.SearchExecuteTyped[FullTextSearch](
//	    ctx, store, FullTextSearch{Query: "quick brown", Limit: 10})
func SearchExecuteTyped[Q any](
	ctx context.Context,
	store *Store,
	input Q,
) ([]SearchResult, error) {
	return executeSliceResult[SearchResult](ctx, store, input)
}

// extractSpatialQuery reads the center coordinates, radius, and limit from a
// query input struct by field name:
//   - X or Lon or Longitude (float64) — center longitude
//   - Y or Lat or Latitude (float64) — center latitude
//   - Radius (float64) — search radius in meters
//   - Limit (int) — maximum results (default 10)
func extractSpatialQuery(input any) (x, y, radius float64, limit int) {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 0, 0, 0, 10
	}

	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		name := field.Name

		switch {
		case (name == "X" || name == "Lon" || name == "Longitude") && field.Type.Kind() == reflect.Float64:
			x = v.Field(i).Float()

		case (name == "Y" || name == "Lat" || name == "Latitude") && field.Type.Kind() == reflect.Float64:
			y = v.Field(i).Float()

		case name == "Radius" && field.Type.Kind() == reflect.Float64:
			radius = v.Field(i).Float()

		case name == "Limit" && field.Type.Kind() == reflect.Int:
			limit = int(v.Field(i).Int())
		}
	}

	if limit <= 0 {
		limit = 10
	}

	return x, y, radius, limit
}

// SpatialExecuteTyped is the type-safe wrapper for spatial range queries.
//
// Usage:
//
//	results, err := metaengine.SpatialExecuteTyped[NearbySearch](
//	    ctx, store, NearbySearch{Lat: 52.5, Lon: 13.4, Radius: 1000, Limit: 10})
func SpatialExecuteTyped[Q any](
	ctx context.Context,
	store *Store,
	input Q,
) ([]SpatialResult, error) {
	return executeSliceResult[SpatialResult](ctx, store, input)
}
