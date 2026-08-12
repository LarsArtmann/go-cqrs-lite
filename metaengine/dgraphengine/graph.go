package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- GraphBackend ---
//
// Dgraph's native strength: O(degree^depth) graph traversal with zero
// degradation. Edges are stored as uid→uid predicates with @reverse for
// bidirectional traversal. GraphAddEdge adds both directions (matching the
// memory engine's symmetric adjacency semantics).

func (e *dgraphEngine) GraphAddEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	fromStr := fmt.Sprint(edge.From)
	toStr := fmt.Sprint(edge.To)

	if err := e.ensureEdgeSchema(ctx, collection); err != nil {
		return err
	}

	pred := graphEdgePredicate(collection)

	// Step 1: Upsert both nodes (create if they don't exist).
	req := &api.Request{CommitNow: true}
	req.Query = fmt.Sprintf(`{
		from_node as var(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s))
		to_node as var(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s))
	}`, dqlString(collection), dqlString(fromStr), dqlString(collection), dqlString(toStr))

	fromJSON, _ := json.Marshal(map[string]any{
		"uid":                  "_:from_new",
		"cqrs.node_collection": collection,
		"cqrs.node_id":         fromStr,
		"dgraph.type":          []string{"GraphNode"},
	})
	toJSON, _ := json.Marshal(map[string]any{
		"uid":                  "_:to_new",
		"cqrs.node_collection": collection,
		"cqrs.node_id":         toStr,
		"dgraph.type":          []string{"GraphNode"},
	})

	req.Mutations = []*api.Mutation{
		{SetJson: fromJSON, Cond: "@if(eq(len(from_node), 0))"},
		{SetJson: toJSON, Cond: "@if(eq(len(to_node), 0))"},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.GraphAddEdge: upsert nodes: %w", err)
	}

	// Step 2: Add bidirectional edges (matches memory engine semantics).
	req2 := &api.Request{CommitNow: true}
	req2.Query = fmt.Sprintf(`{
		from_node as var(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s))
		to_node as var(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s))
	}`, dqlString(collection), dqlString(fromStr), dqlString(collection), dqlString(toStr))

	req2.Mutations = []*api.Mutation{
		{SetNquads: fmt.Appendf(nil, "uid(from_node) <%s> uid(to_node) .", pred)},
		{SetNquads: fmt.Appendf(nil, "uid(to_node) <%s> uid(from_node) .", pred)},
	}

	if _, err := e.client.NewTxn().Do(ctx, req2); err != nil {
		return fmt.Errorf("dgraphengine.GraphAddEdge: add edges: %w", err)
	}

	return nil
}

func (e *dgraphEngine) GraphNeighbors(
	ctx context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

	nodeStr := fmt.Sprint(node)
	pred := graphEdgePredicate(collection)

	var query string
	if depth == 1 {
		query = fmt.Sprintf(`{
			root(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s)) {
				%s { cqrs.node_id }
			}
		}`, dqlString(collection), dqlString(nodeStr), pred)
	} else {
		query = fmt.Sprintf(`{
			root(func: eq(cqrs.node_collection, %s)) @filter(eq(cqrs.node_id, %s)) @recurse(depth: %d, loop: false) {
				cqrs.node_id
				%s
			}
		}`, dqlString(collection), dqlString(nodeStr), depth, pred)
	}

	resp, err := e.client.NewReadOnlyTxn().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.GraphNeighbors: %w", err)
	}

	var root struct {
		Root []map[string]any `json:"root"`
	}

	if err := json.Unmarshal(resp.GetJson(), &root); err != nil {
		return nil, fmt.Errorf("dgraphengine.GraphNeighbors: unmarshal: %w", err)
	}

	if len(root.Root) == 0 {
		return []any{}, nil
	}

	neighbors := extractNeighborIDs(root.Root[0][pred], pred, depth)

	// Deduplicate and exclude the start node (matches memory engine's visited set).
	seen := map[string]bool{nodeStr: true}
	result := make([]any, 0, len(neighbors))

	for _, n := range neighbors {
		key := fmt.Sprint(n)
		if !seen[key] {
			seen[key] = true
			result = append(result, n)
		}
	}

	return result, nil
}

// extractNeighborIDs recursively extracts node IDs from the Dgraph response,
// handling both depth=1 (flat array) and depth>1 (nested @recurse response).
func extractNeighborIDs(v any, pred string, depth int) []any {
	nodes, ok := v.([]any)
	if !ok {
		return nil
	}

	var result []any

	for _, nodeVal := range nodes {
		node, ok := nodeVal.(map[string]any)
		if !ok {
			continue
		}

		if idStr, ok := node["cqrs.node_id"].(string); ok {
			result = append(result, idStr)
		}

		if depth > 1 {
			if nested, ok := node[pred]; ok {
				result = append(result, extractNeighborIDs(nested, pred, depth-1)...)
			}
		}
	}

	return result
}
