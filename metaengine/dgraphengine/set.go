package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

// --- SetBackend ---

func (e *dgraphEngine) SetAdd(ctx context.Context, col string, key any) error {
	keyStr := fmt.Sprint(key)

	req := &api.Request{CommitNow: true}
	req.Query = fmt.Sprintf(`{
		entry as var(func: eq(cqrs.set_collection, %s)) @filter(eq(cqrs.set_key, %s))
	}`, dqlString(col), dqlString(keyStr))

	createJSON, _ := json.Marshal(map[string]any{
		"uid":               "_:new",
		"cqrs.set_collection": col,
		"cqrs.set_key":      keyStr,
		"dgraph.type":       []string{"MetaSetEntry"},
	})

	req.Mutations = []*api.Mutation{
		{SetJson: createJSON, Cond: "@if(eq(len(entry), 0))"},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.SetAdd: %w", err)
	}

	return nil
}

func (e *dgraphEngine) SetContains(ctx context.Context, col string, key any) (bool, error) {
	keyStr := fmt.Sprint(key)

	q := fmt.Sprintf(`{
		entry(func: eq(cqrs.set_collection, %s)) @filter(eq(cqrs.set_key, %s)) {
			count(uid)
		}
	}`, dqlString(col), dqlString(keyStr))

	resp, err := e.client.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return false, fmt.Errorf("dgraphengine.SetContains: %w", err)
	}

	var result struct {
		Entry []struct {
			Count int `json:"count"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return false, fmt.Errorf("dgraphengine.SetContains: unmarshal: %w", err)
	}

	return len(result.Entry) > 0 && result.Entry[0].Count > 0, nil
}
