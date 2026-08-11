package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

// --- MapBackend ---
//
// Models key→value entries via Dgraph upserts: one node per (collection, key)
// pair. MapSet uses a conditional upsert (create-if-absent or update-if-exists).
// MapGet queries by indexed (collection, key). MapDelete nulls all predicates.

func (e *dgraphEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	keyStr := fmt.Sprint(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("dgraphengine.MapSet: marshal: %w", err)
	}

	valueStr := string(data)

	req := &api.Request{CommitNow: true}
	req.Query = `query entry($col: string, $key: string) {
		entry as var(func: eq(cqrs.map_collection, $col)) @filter(eq(cqrs.map_key, $key))
	}`
	req.Vars = map[string]string{"$col": col, "$key": keyStr}

	createJSON, _ := json.Marshal(map[string]any{
		"uid":                 "_:new",
		"cqrs.map_collection": col,
		"cqrs.map_key":        keyStr,
		"cqrs.map_value":      valueStr,
		"dgraph.type":         []string{"MetaMapEntry"},
	})

	updateJSON, _ := json.Marshal(map[string]any{
		"uid":            "uid(entry)",
		"cqrs.map_value": valueStr,
	})

	req.Mutations = []*api.Mutation{
		{SetJson: createJSON, Cond: "@if(eq(len(entry), 0))"},
		{SetJson: updateJSON, Cond: "@if(eq(len(entry), 1))"},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.MapSet: %w", err)
	}

	return nil
}

func (e *dgraphEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	keyStr := fmt.Sprint(key) //art-dupl:accept dgraph key formatting idiom

	q := `query entry($col: string, $key: string) {
		entry(func: eq(cqrs.map_collection, $col)) @filter(eq(cqrs.map_key, $key)) {
			cqrs.map_value
		}
	}`

	resp, err := e.client.NewReadOnlyTxn().
		QueryWithVars(ctx, q, map[string]string{"$col": col, "$key": keyStr})
	if err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: %w", err)
	}

	var result struct {
		Entry []struct {
			MapValue string `json:"cqrs.map_value"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: unmarshal: %w", err)
	}

	if len(result.Entry) == 0 {
		return nil, false, nil
	}

	var val any

	if err := json.Unmarshal([]byte(result.Entry[0].MapValue), &val); err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: decode value: %w", err)
	}

	return val, true, nil
}

func (e *dgraphEngine) MapDelete(ctx context.Context, col string, key any) error {
	keyStr := fmt.Sprint(key) //art-dupl:accept dgraph key formatting idiom

	req := &api.Request{CommitNow: true}
	req.Query = `query entry($col: string, $key: string) {
		entry as var(func: eq(cqrs.map_collection, $col)) @filter(eq(cqrs.map_key, $key))
	}`
	req.Vars = map[string]string{"$col": col, "$key": keyStr}

	deleteJSON, _ := json.Marshal(map[string]any{
		"uid":                 "uid(entry)",
		"cqrs.map_collection": nil,
		"cqrs.map_key":        nil,
		"cqrs.map_value":      nil,
	})

	req.Mutations = []*api.Mutation{
		{DeleteJson: deleteJSON},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.MapDelete: %w", err)
	}

	return nil
}
