package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/internal/testhelpers"
)

func newBenchRegistry() *catalog.Registry {
	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})

	return reg
}

func benchmarkRegistryWithCommand() (*catalog.Registry, *catalog.Catalog) {
	reg := newBenchRegistry()
	testhelpers.AddCommandWithSchema(
		reg,
		"svc",
		"CreateOrder",
		"CreateOrder",
		"1.0.0",
		&catalog.Schema{Type: "object"},
	)

	return reg, reg.Build()
}

func BenchmarkRegistry_Build(b *testing.B) {
	for b.Loop() {
		benchmarkRegistryWithCommand()
	}
}

func BenchmarkSchemaFromType(b *testing.B) {
	type Order struct {
		ID     string `json:"id"`
		Amount int    `json:"amount"`
	}

	for b.Loop() {
		catalog.SchemaFromType[Order]()
	}
}

func BenchmarkAsyncAPI_Export(b *testing.B) {
	_, cat := benchmarkRegistryWithCommand()

	b.ResetTimer()

	for b.Loop() {
		asyncapi.NewExporter("Bench", "1.0.0").Export(cat)
	}
}

func BenchmarkAsyncAPI_MarshalYAML(b *testing.B) {
	_, cat := benchmarkRegistryWithCommand()
	doc := asyncapi.NewExporter("Bench", "1.0.0").Export(cat)

	b.ResetTimer()

	for b.Loop() {
		_, _ = doc.MarshalYAML()
	}
}

func BenchmarkEventCatalog_Export(b *testing.B) {
	reg := newBenchRegistry()

	for i := range 10 {
		reg.AddEvent("svc", catalog.Message{
			Kind:      catalog.EventMessage,
			ID:        "Evt" + string(rune('A'+i)),
			Name:      "Evt" + string(rune('A'+i)),
			Version:   "1.0.0",
			Direction: catalog.Sends,
		})
	}

	cat := reg.Build()

	b.ResetTimer()

	for b.Loop() {
		exp := eventcatalog.NewExporter(b.TempDir())
		_ = exp.Export(cat)
	}
}
