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
	raw, err := store.ExecuteCtx(ctx, input)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, ErrNotFound
	}

	results, ok := raw.([]VectorResult)
	if !ok {
		return nil, errExecuteTypeMismatch
	}

	return results, nil
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
	raw, err := store.ExecuteCtx(ctx, input)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, ErrNotFound
	}

	results, ok := raw.([]SearchResult)
	if !ok {
		return nil, errExecuteTypeMismatch
	}

	return results, nil
}
