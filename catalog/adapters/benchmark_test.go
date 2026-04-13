package adapters_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/internal/testhelpers"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type benchCreateUser struct {
	*command.CatalogCore

	Name  string `json:"name"`
	Email string `json:"email"`
}

func benchCommand() *benchCreateUser {
	aggID := id.NewAggregateID()
	return &benchCreateUser{
		CatalogCore: command.NewCatalogCore("user.create", aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0",
		}),
	}
}

func benchBuilderWithCommand() *adapters.CatalogBuilder {
	builder := adapters.NewBuilder("Bench API", "1.0.0")
	builder.AddService("svc", "Service", "1.0.0", "")
	builder.AddCommand("svc", benchCommand())
	return builder
}

func BenchmarkBuilder_Build(b *testing.B) {
	for b.Loop() {
		benchBuilderWithCommand().Build()
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
	cat := benchBuilderWithCommand().Build()

	b.ResetTimer()

	testhelpers.BenchmarkEventCatalogExport(b, cat)
}
