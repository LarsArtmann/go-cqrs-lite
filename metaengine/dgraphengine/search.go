package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- SearchBackend ---
//
// Dgraph's built-in term index (@index(term)) provides efficient full-text
// search via anyofterms(). The term index tokenizes text on word boundaries,
// enabling case-insensitive matching.

func (e *dgraphEngine) SearchInsert(ctx context.Context, col string, doc metaengine.IndexedText) error {
	req := &api.Request{CommitNow: true}
	req.Query = fmt.Sprintf(`{
		doc as var(func: eq(cqrs.search_collection, %s)) @filter(eq(cqrs.search_id, %s))
	}`, dqlString(col), dqlString(doc.ID))

	createJSON, _ := json.Marshal(map[string]any{
		"uid":                   "_:new",
		"cqrs.search_collection": col,
		"cqrs.search_id":        doc.ID,
		"cqrs.search_content":   doc.Content,
		"dgraph.type":           []string{"SearchDoc"},
	})

	updateJSON, _ := json.Marshal(map[string]any{
		"uid":                 "uid(doc)",
		"cqrs.search_content": doc.Content,
	})

	req.Mutations = []*api.Mutation{
		{SetJson: createJSON, Cond: "@if(eq(len(doc), 0))"},
		{SetJson: updateJSON, Cond: "@if(eq(len(doc), 1))"},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.SearchInsert: %w", err)
	}

	return nil
}

func (e *dgraphEngine) SearchQuery(
	ctx context.Context,
	collection, query string,
	limit int,
) ([]metaengine.SearchResult, error) {
	firstClause := ""
	if limit > 0 {
		firstClause = fmt.Sprintf(", first: %d", limit)
	}

	q := fmt.Sprintf(`{
		docs(func: anyofterms(cqrs.search_content, %s)%s) @filter(eq(cqrs.search_collection, %s)) {
			cqrs.search_id
		}
	}`, dqlString(query), firstClause, dqlString(collection))

	resp, err := e.client.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.SearchQuery: %w", err)
	}

	var result struct {
		Docs []struct {
			SearchID string `json:"cqrs.search_id"`
		} `json:"docs"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.SearchQuery: unmarshal: %w", err)
	}

	out := make([]metaengine.SearchResult, 0, len(result.Docs))
	for _, d := range result.Docs {
		out = append(out, metaengine.SearchResult{ID: d.SearchID})
	}

	return out, nil
}
