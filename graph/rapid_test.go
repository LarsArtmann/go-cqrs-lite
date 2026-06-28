package graph

import (
	"fmt"
	"sort"
	"testing"

	"pgregory.net/rapid"
)

// strKey extracts the string identity of a NodeRef for property test maps.
func strKey(ref NodeRef) string {
	if s, ok := ref.KeyValue.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", ref.KeyValue)
}

// rapidNode is a compact node identity for property tests.

// genGraph generates a random graph with N nodes and M directed edges.
// Returns the seeded MemoryDriver plus the set of all node refs and edge refs.
func genGraph(t *rapid.T, numNodes, numEdges int) (*MemoryDriver, []NodeRef, []EdgeRef) {
	driver := NewMemoryDriver()

	nodes := make([]NodeRef, 0, numNodes)

	for i := 0; i < numNodes; i++ {
		ref := NodeRef{Label: "User", KeyProp: "id", KeyValue: fmt.Sprintf("n%d", i)}
		nodes = append(nodes, ref)
	}

	edges := make([]EdgeRef, 0, numEdges)

	for i := 0; i < numEdges; i++ {
		if numNodes < 2 {
			break
		}

		fromIdx := rapid.IntRange(0, numNodes-1).Draw(t, "from")
		toIdx := rapid.IntRange(0, numNodes-1).Draw(t, "to")

		if fromIdx == toIdx {
			toIdx = (fromIdx + 1) % numNodes
		}

		edge := EdgeRef{
			Type: "KNOWS",
			From: nodes[fromIdx],
			To:   nodes[toIdx],
		}

		edges = append(edges, edge)
	}

	_ = driver.RunInTx(func(sink GraphSink) error {
		for _, n := range nodes {
			_ = sink.MergeNode(n, map[string]any{"id": n.KeyValue})
		}

		for _, e := range edges {
			_ = sink.MergeEdge(e, nil)
		}

		return nil
	})

	return driver, nodes, edges
}

// TestRapid_TraverseNoDuplicates: Traverse results should never contain the
// start node or duplicates.
func TestRapid_TraverseNoDuplicates(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		numNodes := rapid.IntRange(2, 10).Draw(t, "numNodes")
		numEdges := rapid.IntRange(1, 15).Draw(t, "numEdges")

		driver, nodes, _ := genGraph(t, numNodes, numEdges)

		startIdx := rapid.IntRange(0, numNodes-1).Draw(t, "start")
		startNode := nodes[startIdx]
		reachable := driver.Traverse(startNode, "KNOWS", -1)

		seen := make(map[string]bool, len(reachable))

		for _, n := range reachable {
			key := strKey(n.Ref)

			if key == strKey(startNode) {
				t.Fatalf("Traverse result includes start node: %s", key)
			}

			if seen[key] {
				t.Fatalf("Traverse result has duplicate: %s", key)
			}

			seen[key] = true
		}
	})
}

// TestRapid_ShortestPathIsValidPath: if ShortestPath returns a path, every
// consecutive pair in the path must be connected by an edge in the graph.
func TestRapid_ShortestPathIsValidPath(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		numNodes := rapid.IntRange(2, 8).Draw(t, "numNodes")
		numEdges := rapid.IntRange(1, 12).Draw(t, "numEdges")

		driver, nodes, _ := genGraph(t, numNodes, numEdges)

		fromIdx := rapid.IntRange(0, numNodes-1).Draw(t, "from")
		toIdx := rapid.IntRange(0, numNodes-1).Draw(t, "to")

		if fromIdx == toIdx {
			toIdx = (fromIdx + 1) % numNodes
		}

		path, err := driver.ShortestPath(nodes[fromIdx], nodes[toIdx])
		if err != nil {
			return // no path found — that's valid
		}

		// Verify path is non-empty and endpoints match.
		if len(path) < 2 {
			t.Fatalf("path too short: %d hops", len(path))
		}

		if path[0] != nodes[fromIdx] {
			t.Fatalf("path start mismatch: %v != %v", path[0], nodes[fromIdx])
		}

		if path[len(path)-1] != nodes[toIdx] {
			t.Fatalf("path end mismatch: %v != %v", path[len(path)-1], nodes[toIdx])
		}

		// Every consecutive pair must be connected.
		data := driver.Snapshot()

		for i := 0; i < len(path)-1; i++ {
			fromStr := strKey(path[i])
			toStr := strKey(path[i+1])

			connected := false

			for ek := range data.edges {
				ekFromStr := ek.from.keyVal
				ekToStr := ek.to.keyVal

				if (ekFromStr == fromStr && ekToStr == toStr) ||
					(ekFromStr == toStr && ekToStr == fromStr) {
					connected = true

					break
				}
			}

			if !connected {
				t.Fatalf("path has disconnected hop at step %d: %v → %v", i, path[i], path[i+1])
			}
		}
	})
}

// TestRapid_QueryLabelFilter: Query with a label filter should only return
// nodes matching that label, never nodes of other labels.
func TestRapid_QueryLabelFilter(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		numNodes := rapid.IntRange(1, 10).Draw(t, "numNodes")

		driver := NewMemoryDriver()
		_ = driver.RunInTx(func(sink GraphSink) error {
			for i := 0; i < numNodes; i++ {
				label := "User"
				if i%2 == 0 {
					label = "Bot"
				}

				ref := NodeRef{Label: label, KeyProp: "id", KeyValue: fmt.Sprintf("n%d", i)}
				_ = sink.MergeNode(ref, map[string]any{"id": ref.KeyValue})
			}

			return nil
		})

		users := driver.Query(Pattern{Label: "User"})
		bots := driver.Query(Pattern{Label: "Bot"})

		for _, n := range users {
			if n.Ref.Label != "User" {
				t.Fatalf("Query(User) returned non-User: %v", n.Ref)
			}
		}

		for _, n := range bots {
			if n.Ref.Label != "Bot" {
				t.Fatalf("Query(Bot) returned non-Bot: %v", n.Ref)
			}
		}

		// No overlap between User and Bot sets.
		userSet := make(map[string]bool, len(users))

		for _, n := range users {
			userSet[strKey(n.Ref)] = true
		}

		for _, n := range bots {
			if userSet[strKey(n.Ref)] {
				t.Fatalf("node %s appeared in both User and Bot query results", n.Ref.KeyValue)
			}
		}
	})
}

// TestRapid_NeighborsSymmetricDegree: the Neighbors of node B should include
// every node directly connected to B, regardless of edge direction.
func TestRapid_NeighborsComplete(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		numNodes := rapid.IntRange(3, 8).Draw(t, "numNodes")

		driver, nodes, edges := genGraph(t, numNodes, rapid.IntRange(2, 15).Draw(t, "numEdges"))

		// Pick any non-start node.
		pickIdx := rapid.IntRange(0, numNodes-1).Draw(t, "pick")
		center := nodes[pickIdx]

		// Compute expected neighbors.
		expected := make(map[string]bool)

		for _, e := range edges {
			if strKey(e.From) == strKey(center) {
				expected[strKey(e.To)] = true
			}

			if strKey(e.To) == strKey(center) {
				expected[strKey(e.From)] = true
			}
		}

		neighborNodes, _ := driver.Neighbors(center)

		actual := make([]string, 0, len(neighborNodes))

		for _, n := range neighborNodes {
			actual = append(actual, strKey(n.Ref))
		}

		sort.Strings(actual)

		for _, n := range neighborNodes {
			if !expected[strKey(n.Ref)] && strKey(n.Ref) != strKey(center) {
				t.Fatalf("Neighbors(%s) returned unexpected node: %s", center.KeyValue, n.Ref.KeyValue)
			}
		}
	})
}
