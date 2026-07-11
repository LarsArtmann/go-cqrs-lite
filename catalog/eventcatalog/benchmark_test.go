package eventcatalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

func BenchmarkEventCatalog_Export(b *testing.B) {
	b.ReportAllocs()

	reg := catalog.NewRegistry("BenchCatalog", "1.0.0")
	reg.AddService(catalog.Service{ID: "svc", Name: "Service", Version: "1.0.0"})

	for i := range 10 {
		reg.AddEvent("svc", catalog.Message{
			Kind:      catalog.EventMessage,
			ID:        catalog.MessageID("Evt" + string(rune('A'+i))),
			Name:      catalog.Name("Evt" + string(rune('A'+i))),
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
