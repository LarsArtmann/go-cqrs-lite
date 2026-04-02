package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

func BenchmarkRegistry_Build(b *testing.B) {
	for b.Loop() {
		reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
		reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
		reg.AddCommand("svc", catalog.Message{
			Kind:    catalog.CommandMessage,
			ID:      "CreateOrder",
			Name:    "CreateOrder",
			Version: "1.0.0",
			Schema:  &catalog.Schema{Type: "object"},
		})
		reg.Build()
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
	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	for i := range 10 {
		reg.AddCommand("svc", catalog.Message{
			Kind:    catalog.CommandMessage,
			ID:      "Cmd" + string(rune('A'+i)),
			Name:    "Cmd" + string(rune('A'+i)),
			Version: "1.0.0",
			Schema:  &catalog.Schema{Type: "object"},
		})
	}
	cat := reg.Build()

	b.ResetTimer()
	for b.Loop() {
		asyncapi.NewExporter("Bench", "1.0.0").Export(cat)
	}
}

func BenchmarkAsyncAPI_MarshalYAML(b *testing.B) {
	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	reg.AddCommand("svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "CreateOrder", Version: "1.0.0",
		Schema: &catalog.Schema{Type: "object", Properties: map[string]catalog.Property{"id": {Type: "string"}}},
	})
	cat := reg.Build()
	doc := asyncapi.NewExporter("Bench", "1.0.0").Export(cat)

	b.ResetTimer()
	for b.Loop() {
		doc.MarshalYAML()
	}
}

func BenchmarkEventCatalog_Export(b *testing.B) {
	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
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
		exp.Export(cat)
	}
}
