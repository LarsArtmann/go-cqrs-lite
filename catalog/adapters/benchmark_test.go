package adapters_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type benchCreateUser struct {
	*command.CatalogCore

	Name  string `json:"name"`
	Email string `json:"email"`
}

func BenchmarkBuilder_Build(b *testing.B) {
	aggID := id.NewAggregateID()

	for b.Loop() {
		builder := adapters.NewBuilder("Bench API", "1.0.0")
		builder.AddService("svc", "Service", "1.0.0", "")
		builder.AddCommand("svc", &benchCreateUser{
			CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
				Name: "CreateUser", Version: "1.0.0",
			}),
		})
		builder.Build()
	}
}

func BenchmarkBuilder_FromCommandDispatcher(b *testing.B) {
	d := command.NewDispatcher()
	for i := range 10 {
		d.RegisterCatalogEntry(
			command.Type(fmt.Sprintf("cmd.%d", i)),
			command.CatalogMeta{Name: fmt.Sprintf("Cmd%d", i), Version: "1.0.0"},
		)
	}

	b.ResetTimer()

	for b.Loop() {
		builder := adapters.NewBuilder("Bench", "1.0.0")
		builder.AddService("svc", "Service", "1.0.0", "")
		adapters.FromCommandDispatcher(builder, "svc", d)
		builder.Build()
	}
}

func BenchmarkBuilder_ExportEventCatalog(b *testing.B) {
	aggID := id.NewAggregateID()

	builder := adapters.NewBuilder("Bench API", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "")
	builder.AddCommand("svc", &benchCreateUser{
		CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0",
		}),
	})
	cat := builder.Build()

	b.ResetTimer()

	for b.Loop() {
		exp := eventcatalog.NewExporter(b.TempDir())
		_ = exp.Export(cat)
	}
}
