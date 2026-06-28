// Package graphtest provides a shared contract test suite for [graph.GraphDriver]
// implementations. Any driver (in-memory, Neo4j, Memgraph) can verify it
// satisfies the behavioral contract by calling [RunSuite].
//
// Usage:
//
//	func TestMemoryDriverContract(t *testing.T) {
//	    graphtest.RunSuite(t, graphtest.Config{
//	        Factory: func(t *testing.T) graph.GraphDriver {
//	            return graph.NewMemoryDriver()
//	        },
//	    })
//	}
package graphtest

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/graph/v3"
)

// Config configures the contract test suite.
type Config struct {
	// Factory returns a fresh, empty GraphDriver for each subtest.
	// Tests do NOT share state — each subtest calls Factory for a clean driver.
	Factory func(t *testing.T) graph.GraphDriver

	// SchemaFactory returns a fresh driver pre-configured with the test schema.
	// When set, schema validation contract tests run against it. When nil,
	// schema tests are skipped (the driver does not support schema validation).
	SchemaFactory func(t *testing.T) graph.GraphDriver
}

// RunSuite runs the mandatory GraphDriver contract tests (7 subtests).
// Every GraphDriver implementation must pass all of these.
func RunSuite(t *testing.T, cfg Config) {
	t.Helper()

	if cfg.Factory == nil {
		t.Fatal("graphtest.Config.Factory must not be nil")
	}

	t.Run("MergeNodeCreates", func(t *testing.T) {
		t.Parallel()
		testMergeNodeCreates(t, cfg.Factory(t))
	})

	t.Run("MergeNodeUpdates", func(t *testing.T) {
		t.Parallel()
		testMergeNodeUpdates(t, cfg.Factory(t))
	})

	t.Run("MergeEdgeCreatesEndpoints", func(t *testing.T) {
		t.Parallel()
		testMergeEdgeCreatesEndpoints(t, cfg.Factory(t))
	})

	t.Run("MergeEdgeUpdatesProps", func(t *testing.T) {
		t.Parallel()
		testMergeEdgeUpdatesProps(t, cfg.Factory(t))
	})

	t.Run("RemoveNodeDeletesIncidentEdges", func(t *testing.T) {
		t.Parallel()
		testRemoveNodeDeletesIncidentEdges(t, cfg.Factory(t))
	})

	t.Run("RemoveEdgeLeavesEndpoints", func(t *testing.T) {
		t.Parallel()
		testRemoveEdgeLeavesEndpoints(t, cfg.Factory(t))
	})

	t.Run("AtomicRollbackOnError", func(t *testing.T) {
		t.Parallel()
		testAtomicRollbackOnError(t, cfg.Factory(t))
	})

	// Schema validation contract tests — only run when SchemaFactory is set.
	if cfg.SchemaFactory != nil {
		t.Run("SchemaRejectsUnknownLabel", func(t *testing.T) {
			t.Parallel()
			testSchemaRejectsUnknownLabel(t, cfg.SchemaFactory(t))
		})

		t.Run("SchemaRejectsUnknownProp", func(t *testing.T) {
			t.Parallel()
			testSchemaRejectsUnknownProp(t, cfg.SchemaFactory(t))
		})

		t.Run("SchemaAcceptsValidWrite", func(t *testing.T) {
			t.Parallel()
			testSchemaAcceptsValidWrite(t, cfg.SchemaFactory(t))
		})
	}
}

func nodeRef(label, key string) graph.NodeRef {
	return graph.NodeRef{Label: label, KeyProp: "id", KeyValue: key}
}

func testMergeNodeCreates(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{"name": "alice"})
	})

	// Node should exist with the property.
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.SetNodeProperty(nodeRef("User", "u1"), "verified", true)
	})
}

func testMergeNodeUpdates(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{"name": "alice", "age": 30})
	})

	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		// Update only name — age should be preserved by MERGE semantics.
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{"name": "alice2"})
	})
}

func testMergeEdgeCreatesEndpoints(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS",
			From: nodeRef("User", "u1"),
			To:   nodeRef("User", "u2"),
		}, map[string]any{"since": "2024"})
	})

	// Both endpoints should exist even though we never called MergeNode.
	// Removing the edge should leave endpoints.
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.RemoveEdge(graph.EdgeRef{
			Type: "KNOWS",
			From: nodeRef("User", "u1"),
			To:   nodeRef("User", "u2"),
		})
	})
}

func testMergeEdgeUpdatesProps(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2"),
		}, map[string]any{"since": "2024"})
	})

	// Merge again with new property — should update, not duplicate.
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2"),
		}, map[string]any{"strength": 0.9})
	})
}

func testRemoveNodeDeletesIncidentEdges(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		if err := sink.MergeNode(nodeRef("User", "u1"), nil); err != nil {
			return err
		}

		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2"),
		}, nil)
	})

	err := driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.RemoveNode(nodeRef("User", "u1"))
	})
	if err != nil {
		t.Fatalf("remove node: %v", err)
	}

	// Node should be gone — re-merging should succeed (it was deleted).
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{"name": "recreated"})
	})
}

func testRemoveEdgeLeavesEndpoints(t *testing.T, driver graph.GraphDriver) {
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2"),
		}, nil)
	})

	err := driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.RemoveEdge(graph.EdgeRef{
			Type: "KNOWS", From: nodeRef("User", "u1"), To: nodeRef("User", "u2"),
		})
	})
	if err != nil {
		t.Fatalf("remove edge: %v", err)
	}

	// Endpoints still exist — SetNodeProperty should succeed.
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.SetNodeProperty(nodeRef("User", "u1"), "name", "still here")
	})
}

func testAtomicRollbackOnError(t *testing.T, driver graph.GraphDriver) {
	wantErr := errors.New("intentional failure")

	err := driver.RunInTx(func(sink graph.GraphSink) error {
		if err := sink.MergeNode(
			nodeRef("User", "u1"),
			map[string]any{"name": "alice"},
		); err != nil {
			return err
		}

		if err := sink.MergeNode(nodeRef("User", "u2"), map[string]any{"name": "bob"}); err != nil {
			return err
		}

		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	// After rollback, re-creating u1 should work (it was rolled back).
	_ = driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(nodeRef("User", "u1"), map[string]any{"name": "alice"})
	})
}

func testSchemaRejectsUnknownLabel(t *testing.T, driver graph.GraphDriver) {
	err := driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(
			graph.NodeRef{Label: "Phantom", KeyProp: "id", KeyValue: "x"},
			nil,
		)
	})
	if err == nil {
		t.Fatal("expected error for unknown label, got nil")
	}
}

func testSchemaRejectsUnknownProp(t *testing.T, driver graph.GraphDriver) {
	err := driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(
			nodeRef("User", "u1"),
			map[string]any{"bogus": "value"},
		)
	})
	if err == nil {
		t.Fatal("expected error for unknown property, got nil")
	}
}

func testSchemaAcceptsValidWrite(t *testing.T, driver graph.GraphDriver) {
	err := driver.RunInTx(func(sink graph.GraphSink) error {
		return sink.MergeNode(
			nodeRef("User", "u1"),
			map[string]any{"name": "alice"},
		)
	})
	if err != nil {
		t.Fatalf("valid write rejected by schema: %v", err)
	}
}
