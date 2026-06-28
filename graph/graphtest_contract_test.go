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
	})
}
