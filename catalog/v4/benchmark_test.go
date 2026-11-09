package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func newBenchRegistry() *catalog.Registry {
	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})

	return reg
}

func benchmarkRegistryWithCommand(tb testing.TB) *catalog.Catalog {
	reg := newBenchRegistry()
	cattest.AddCommandWithSchema(
		tb,
		reg,
		"svc",
		"CreateOrder",
		"CreateOrder",
		"1.0.0",
		&catalog.Schema{Type: catalog.TypeObject},
	)

	return reg.Build()
}

func BenchmarkRegistry_Build(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkRegistryWithCommand(b)
	}
}

func BenchmarkSchemaFromType(b *testing.B) {
	b.ReportAllocs()
	type Order struct {
		ID     string `json:"id"`
		Amount int    `json:"amount"`
	}

	for b.Loop() {
		catalog.SchemaFromType[Order]()
	}
}
