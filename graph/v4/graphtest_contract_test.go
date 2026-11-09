package graph_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/graph/v4"
	"github.com/larsartmann/go-cqrs-lite/graph/v4/graphtest"
)

func TestMemoryDriverContract(t *testing.T) {
	graphtest.RunSuite(t, graphtest.Config{
		Factory: func(t *testing.T) graph.GraphDriver {
			return graph.NewMemoryDriver()
		},
		SchemaFactory: func(t *testing.T) graph.GraphDriver {
			return graph.NewMemoryDriver(graph.WithDriverSchema(contractSchema()))
		},
		ReadableFactory: func(t *testing.T) graph.ReadableDriver {
			driver := graph.NewMemoryDriver()
			graphtest.SeedReadGraph(t, driver)

			return driver
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
