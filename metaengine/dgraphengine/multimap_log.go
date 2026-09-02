package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"
	"time"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

// --- MultimapBackend ---
//
// Models one-to-many key→values via one Dgraph node per (key, value) pair.
// MultiAdd inserts a new node; MultiGet queries all nodes matching the key.
// The value is JSON-serialized for storage (same pattern as MapBackend).

func (e *dgraphEngine) MultiAdd(
	ctx context.Context,
	col string,
	key any,
	value any,
) error {
	keyStr := fmt.Sprint(key)
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("dgraphengine.MultiAdd: marshal value: %w", err)
	}

	data, _ := json.Marshal(map[string]any{
		"uid":                      "_:new",
		"cqrs.multimap_collection": col,
		"cqrs.multimap_key":        keyStr,
		"cqrs.multimap_value":      string(valueJSON),
		"dgraph.type":              []string{"MultimapEntry"},
	})

	if _, err := e.doMutate(ctx, &api.Mutation{
		SetJson: data,
	}); err != nil {
		return fmt.Errorf("dgraphengine.MultiAdd: %w", err)
	}

	return nil
}

func (e *dgraphEngine) MultiGet(
	ctx context.Context,
	col string,
	key any,
) ([]any, error) {
	keyStr := fmt.Sprint(key)

	q := `query mm($col: string, $key: string) {
		entries(func: eq(cqrs.multimap_collection, $col)) @filter(eq(cqrs.multimap_key, $key)) {
			cqrs.multimap_value
		}
	}`

	return e.queryValueEntries(ctx, "MultiGet", q,
		map[string]string{"$col": col, "$key": keyStr}, "cqrs.multimap_value")
}

// queryValueEntries runs q and decodes each entry's predicate value (a JSON
// document stored as a string) in entry order. op names the caller in errors.
func (e *dgraphEngine) queryValueEntries(
	ctx context.Context,
	op, q string,
	vars map[string]string,
	pred string,
) ([]any, error) {
	resp, err := e.readTx().QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.%s: %w", op, err)
	}

	var result struct {
		Entries []map[string]string `json:"entries"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.%s: unmarshal: %w", op, err)
	}

	out := make([]any, 0, len(result.Entries))
	for _, entry := range result.Entries {
		var v any
		if err := json.Unmarshal([]byte(entry[pred]), &v); err != nil {
			return nil, fmt.Errorf("dgraphengine.%s: decode value: %w", op, err)
		}

		out = append(out, v)
	}

	return out, nil
}

// --- LogBackend ---
//
// Models append-only ordered logs via Dgraph nodes with a nanosecond
// timestamp as the sequence. LogTail queries ordered by timestamp descending.

func (e *dgraphEngine) LogAppend(ctx context.Context, col string, value any) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("dgraphengine.LogAppend: marshal value: %w", err)
	}

	data, _ := json.Marshal(map[string]any{
		"uid":                 "_:new",
		"cqrs.log_collection": col,
		"cqrs.log_seq":        time.Now().UnixNano(),
		"cqrs.log_value":      string(valueJSON),
		"dgraph.type":         []string{"LogEntry"},
	})

	if _, err := e.doMutate(ctx, &api.Mutation{
		SetJson: data,
	}); err != nil {
		return fmt.Errorf("dgraphengine.LogAppend: %w", err)
	}

	return nil
}

func (e *dgraphEngine) LogTail(ctx context.Context, col string, limit int) ([]any, error) {
	// DQL's `first:` is a query-syntax element, not a value, so it cannot be
	// passed as a $variable in QueryWithVars. limit is an int — %d emits only
	// digits, so this is injection-safe by construction.
	firstClause := ""
	if limit > 0 {
		firstClause = fmt.Sprintf(", first: %d", limit)
	}

	q := fmt.Sprintf(`query log($col: string) {
		entries(func: eq(cqrs.log_collection, $col), orderdesc: cqrs.log_seq%s) {
			cqrs.log_value
		}
	}`, firstClause)

	out, err := e.queryValueEntries(ctx, "LogTail", q,
		map[string]string{"$col": col}, "cqrs.log_value")
	if err != nil {
		return nil, err
	}

	// Reverse to chronological order (oldest first), matching memory engine.
	slices.Reverse(out)

	return out, nil
}
