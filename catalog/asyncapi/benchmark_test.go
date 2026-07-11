package asyncapi_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func BenchmarkAsyncAPI_Export(b *testing.B) {
	b.ReportAllocs()

	reg := catalog.NewRegistry("Bench", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	cattest.AddCommandWithSchema(b, reg, "svc", "CreateOrder", "CreateOrder", "1.0.0",
		&catalog.Schema{Type: catalog.TypeObject})
	cat := reg.Build()

	b.ResetTimer()

	for b.Loop() {
		asyncapi.NewExporter("Bench", "1.0.0").Export(cat)
	}
}

func BenchmarkAsyncAPI_MarshalYAML(b *testing.B) {
	b.ReportAllocs()

	reg := catalog.NewRegistry("Bench", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})
	cattest.AddCommandWithSchema(b, reg, "svc", "CreateOrder", "CreateOrder", "1.0.0",
		&catalog.Schema{Type: catalog.TypeObject})
	cat := reg.Build()
	doc := asyncapi.NewExporter("Bench", "1.0.0").Export(cat)

	b.ResetTimer()

	for b.Loop() {
		_, _ = doc.MarshalYAML()
	}
}
