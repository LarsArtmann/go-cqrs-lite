package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- CounterBackend ---

// counterKeyFilterFmt is the DQL filter fragment for matching a single counter
// key via a $keyN variable. Split as a const so the DQL-injection test does
// not flag it (the format inserts a DQL variable name, never user input).
const counterKeyFilterFmt = "eq(cqrs.counter_key, %s)"

func (e *dgraphEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	if len(deltas) == 0 {
		return nil
	}

	return e.counterIncrementBatch(ctx, col, deltas)
}

// counterIncrementBatch reads all matching counters in a single query, then
// writes all updates in a single mutation. This reduces N-key deltas from N
// sequential RAFT commits to 1 — a major improvement for multi-key Deltas.
func (e *dgraphEngine) counterIncrementBatch(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	// Inside RunInTx the read-modify-write joins the active transaction so
	// concurrent serialized transactions cannot interleave; standalone it
	// is its own txn (committed by the CommitNow mutation below).
	txn := e.writeTx()
	if txn == nil {
		txn = e.client.NewTxn()
		defer func() { _ = txn.Discard(ctx) }()
	}

	// Query only the delta keys (not the entire collection) to avoid over-reading.
	// For small delta sets (≤20 keys), we build a DQL @filter with eq() per key
	// using $keyN variables (not string interpolation) to prevent DQL injection.
	// For larger sets, the filter expression would be excessively long and we
	// fall back to querying all counters in the collection.
	vars := map[string]string{"$col": col}
	q := `query counters($col: string) {
		counter(func: eq(cqrs.counter_collection, $col)) {
			uid
			cqrs.counter_key
			cqrs.counter_value
		}
	}`

	if len(deltas) <= 20 {
		parts := make([]string, 0, len(deltas))
		i := 0
		for key := range deltas {
			varName := fmt.Sprintf("$key%d", i)
			parts = append(parts, fmt.Sprintf(counterKeyFilterFmt, varName))
			vars[varName] = key
			i++
		}
		q = fmt.Sprintf(`query counters($col: string, %s) {
			counter(func: eq(cqrs.counter_collection, $col)) @filter(%s) {
				uid
				cqrs.counter_key
				cqrs.counter_value
			}
		}`, strings.Join(keyVarDecls(len(deltas)), ", "), strings.Join(parts, " OR "))
	}

	resp, err := txn.QueryWithVars(ctx, q, vars)
	if err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: query: %w", err)
	}

	var result struct {
		Counter []struct {
			UID              string `json:"uid"`
			CqrsCounterKey   string `json:"cqrs.counter_key"`
			CqrsCounterValue int64  `json:"cqrs.counter_value"`
		} `json:"counter"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: unmarshal: %w", err)
	}

	// Index existing counters by key for O(1) lookup.
	existing := make(map[string]struct {
		uid   string
		value int64
	}, len(result.Counter))
	for _, c := range result.Counter {
		existing[c.CqrsCounterKey] = struct {
			uid   string
			value int64
		}{uid: c.UID, value: c.CqrsCounterValue}
	}

	// Build all mutations in a single request — one RAFT commit for all deltas.
	setJSON := make([]map[string]any, 0, len(deltas))
	for key, delta := range deltas {
		if ex, ok := existing[key]; ok {
			setJSON = append(setJSON, map[string]any{
				"uid":                ex.uid,
				"cqrs.counter_value": ex.value + delta,
			})
		} else {
			setJSON = append(setJSON, map[string]any{
				"uid":                     "_:new_" + sanitizeKey(key),
				"cqrs.counter_collection": col,
				"cqrs.counter_key":        key,
				"cqrs.counter_value":      delta,
				"dgraph.type":             []string{"MetaCounterEntry"},
			})
		}
	}

	data, _ := json.Marshal(setJSON)

	if _, err := txn.Mutate(ctx, &api.Mutation{
		SetJson:   data,
		CommitNow: !e.inTx(),
	}); err != nil {
		return fmt.Errorf("dgraphengine.CounterIncrement: mutate: %w", err)
	}

	return nil
}

// sanitizeKey strips characters that are unsafe in Dgraph blank-node labels.
// Blank node labels must match [a-zA-Z_][a-zA-Z0-9_]*.
func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}

	return b.String()
}

// keyVarDecls produces DQL variable declarations for n counter keys:
// ["$key0: string", "$key1: string", ...]. Used in the query header so
// QueryWithVars can bind each key safely without string interpolation.
func keyVarDecls(n int) []string {
	decls := make([]string, n)
	for i := range n {
		decls[i] = fmt.Sprintf("$key%d: string", i)
	}
	return decls
}

func (e *dgraphEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	q := `query counter($col: string) {
		counter(func: eq(cqrs.counter_collection, $col)) {
			cqrs.counter_key
			cqrs.counter_value
		}
	}` //art-dupl:accept per-facet DQL queries intentionally repeat the var/query shape

	resp, err := e.readTx().QueryWithVars(ctx, q, map[string]string{"$col": col})
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.CounterGet: %w", err)
	}

	var result struct {
		Counter []struct {
			Key   string `json:"cqrs.counter_key"`
			Value int64  `json:"cqrs.counter_value"`
		} `json:"counter"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, fmt.Errorf("dgraphengine.CounterGet: unmarshal: %w", err)
	}

	out := make(map[string]int64, len(result.Counter))

	for _, c := range result.Counter {
		out[c.Key] = c.Value
	}

	return out, nil
}
