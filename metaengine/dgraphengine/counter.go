package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- CounterBackend ---

func (e *dgraphEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	if len(deltas) == 0 {
		return nil
	}

	for key, delta := range deltas {
		if err := e.counterIncrementOne(ctx, col, key, delta); err != nil {
			return err
		}
	}

	return nil
}

// counterIncrementOne atomically increments a single counter key using a
// read-modify-write within a single Dgraph transaction.
func (e *dgraphEngine) counterIncrementOne(
	ctx context.Context,
	col, key string,
	delta int64,
) error {
	txn := e.client.NewTxn()
	defer func() { _ = txn.Discard(ctx) }()

	q := fmt.Sprintf(`{
		counter(func: eq(cqrs.counter_collection, %s)) @filter(eq(cqrs.counter_key, %s)) {
			uid
			cqrs.counter_value
		}
	}`, dqlString(col), dqlString(key))

	resp, err := txn.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: query: %w", err)
	}

	var result struct {
		Counter []struct {
			UID             string `json:"uid"`
			CqrsCounterValue int64  `json:"cqrs.counter_value"`
		} `json:"counter"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: unmarshal: %w", err)
	}

	newValue := delta
	node := map[string]any{
		"cqrs.counter_collection": col,
		"cqrs.counter_key":        key,
		"cqrs.counter_value":      newValue,
		"dgraph.type":             []string{"MetaCounterEntry"},
	}

	if len(result.Counter) > 0 {
		node["uid"] = result.Counter[0].UID
		node["cqrs.counter_value"] = result.Counter[0].CqrsCounterValue + delta
	} else {
		node["uid"] = "_:new"
	}

	data, _ := json.Marshal(node)

	if _, err := txn.Mutate(ctx, &api.Mutation{
		SetJson:   data,
		CommitNow: true,
	}); err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: mutate: %w", err)
	}

	return nil
}

func (e *dgraphEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	q := fmt.Sprintf(`{
		counter(func: eq(cqrs.counter_collection, %s)) {
			cqrs.counter_key
			cqrs.counter_value
		}
	}`, dqlString(col))

	resp, err := e.client.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.CounterGet: %w", err)
	}

	var result struct {
		Counter []struct {
			Key   string `json:"cqrs.counter_key"`
			Value int64  `json:"cqrs.counter_value"`
		} `json:"counter"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.CounterGet: unmarshal: %w", err)
	}

	out := make(map[string]int64, len(result.Counter))

	for _, c := range result.Counter {
		out[c.Key] = c.Value
	}

	return out, nil
}
