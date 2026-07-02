package graph

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Compile-time assertion: MemoryDriver implements ReadableDriver.
var _ ReadableDriver = (*MemoryDriver)(nil)

// Query returns all nodes matching the pattern. See [Pattern] for filter
// semantics. The returned NodeViews contain defensive copies of properties.
func (d *MemoryDriver) Query(p Pattern) []NodeView {
	data := d.Snapshot()

	result := make([]NodeView, 0)

	for k, n := range data.nodes {
		if p.Label != "" && k.label != p.Label {
			continue
		}

		props := copyProps(n.props)

		if p.Where != nil && !p.Where(props) {
			continue
		}

		result = append(result, NodeView{
			Ref:   keyToRef(k),
			Props: props,
		})
	}

	return result
}

// Traverse performs a breadth-first traversal from the given node, following
// edges of the specified type. maxDepth 0 returns immediate neighbors;
// maxDepth < 0 means unlimited. The start node is excluded from results.
func (d *MemoryDriver) Traverse(from NodeRef, via string, maxDepth int) []NodeView {
	data := d.Snapshot()

	fromKey, err := from.key()
	if err != nil {
		return nil
	}

	if _, exists := data.nodes[fromKey]; !exists {
		return nil
	}

	visited := map[nodeKey]struct{}{fromKey: {}}
	queue := []nodeKey{fromKey}
	result := make([]NodeView, 0)

	for depth := 0; len(queue) > 0 && (maxDepth < 0 || depth <= maxDepth); depth++ {
		var next []nodeKey

		for _, current := range queue {
			for edgeKey := range data.edges {
				if edgeKey.typ != via {
					continue
				}

				var neighbor nodeKey

				if edgeKey.from == current {
					neighbor = edgeKey.to
				} else {
					continue
				}

				if _, seen := visited[neighbor]; seen {
					continue
				}

				visited[neighbor] = struct{}{}
				next = append(next, neighbor)

				if n, ok := data.nodes[neighbor]; ok {
					result = append(result, NodeView{
						Ref:   keyToRef(neighbor),
						Props: copyProps(n.props),
					})
				}
			}
		}

		queue = next
	}

	return result
}

// Neighbors returns all nodes and edges directly connected to the given node,
// in both incoming and outgoing directions.
func (d *MemoryDriver) Neighbors(of NodeRef) ([]NodeView, []EdgeView) {
	data := d.Snapshot()

	centerKey, err := of.key()
	if err != nil {
		return nil, nil
	}

	if _, exists := data.nodes[centerKey]; !exists {
		return nil, nil
	}

	neighborSet := make(map[nodeKey]struct{})

	var nodes []NodeView

	for edgeKey, e := range data.edges {
		var (
			neighbor  nodeKey
			connected bool
		)

		if edgeKey.from == centerKey {
			neighbor = edgeKey.to
			connected = true
		} else if edgeKey.to == centerKey {
			neighbor = edgeKey.from
			connected = true
		}

		if !connected {
			continue
		}

		if _, seen := neighborSet[neighbor]; !seen {
			neighborSet[neighbor] = struct{}{}

			if n, ok := data.nodes[neighbor]; ok {
				nodes = append(nodes, NodeView{
					Ref:   keyToRef(neighbor),
					Props: copyProps(n.props),
				})
			}
		}

		_ = e // edge props handled in the edges slice below
	}

	var edges []EdgeView

	for edgeKey, e := range data.edges {
		if edgeKey.from == centerKey || edgeKey.to == centerKey {
			edges = append(edges, EdgeView{
				Ref:   edgeKeyToRef(edgeKey),
				Props: copyProps(e.props),
			})
		}
	}

	return nodes, edges
}

// ShortestPath finds the shortest path between two nodes via BFS.
// Returns the path as an ordered slice of NodeRefs (including from and to).
// Returns ErrPathNotFound if no path exists.
func (d *MemoryDriver) ShortestPath(from, target NodeRef) ([]NodeRef, error) {
	data := d.Snapshot()

	fromKey, err := from.key()
	if err != nil {
		return nil, event.WrapRejection(err, "graph.shortest_path_from",
			"parse shortest path from")
	}

	targetKey, err := target.key()
	if err != nil {
		return nil, event.WrapRejection(err, "graph.shortest_path_to",
			"parse shortest path to")
	}

	if fromKey == targetKey {
		return []NodeRef{from}, nil
	}

	if _, exists := data.nodes[fromKey]; !exists {
		return nil, ErrPathNotFound
	}

	if _, exists := data.nodes[targetKey]; !exists {
		return nil, ErrPathNotFound
	}

	parent := map[nodeKey]nodeKey{fromKey: fromKey}
	queue := []nodeKey{fromKey}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == targetKey {
			return reconstructPath(parent, current), nil
		}

		for edgeKey := range data.edges {
			var (
				neighbor  nodeKey
				connected bool
			)

			if edgeKey.from == current {
				neighbor = edgeKey.to
				connected = true
			} else if edgeKey.to == current {
				neighbor = edgeKey.from
				connected = true
			}

			if !connected {
				continue
			}

			if _, seen := parent[neighbor]; seen {
				continue
			}

			parent[neighbor] = current
			queue = append(queue, neighbor)
		}
	}

	return nil, ErrPathNotFound
}

func reconstructPath(parent map[nodeKey]nodeKey, end nodeKey) []NodeRef {
	var path []nodeKey

	for k := end; ; k = parent[k] {
		path = append([]nodeKey{k}, path...)
		if k == parent[k] {
			break
		}
	}

	refs := make([]NodeRef, len(path))

	for i, k := range path {
		refs[i] = keyToRef(k)
	}

	return refs
}

func keyToRef(k nodeKey) NodeRef {
	return NodeRef{Label: k.label, KeyProp: k.keyProp, KeyValue: k.keyVal}
}

func edgeKeyToRef(k edgeKey) EdgeRef {
	return EdgeRef{Type: k.typ, From: keyToRef(k.from), To: keyToRef(k.to)}
}
