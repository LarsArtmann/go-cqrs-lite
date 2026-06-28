package graph_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/graph/v3/graphtest"
)

func TestMemoryDriverContract(t *testing.T) {
	graphtest.RunSuite(t, graphtest.Config{
		Factory: func(t *testing.T) graph.GraphDriver {
			return graph.NewMemoryDriver()
		},
		SchemaFactory: func(t *testing.T) graph.GraphDriver {
			return graph.NewMemoryDriver(graph.WithDriverSchema(contractSchema()))
		},
	})
}

func contractSchema() *graph.Schema {
	return &graph.Schema{
		Nodes: []graph.NodeType{
			{Label: "User", KeyProp: "id", Properties: []graph.PropertyType{
				{Name: "name"},
			}},
		},
	}
}
