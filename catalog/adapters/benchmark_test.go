package adapters_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
)

type benchCreateUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func benchBuilderWithCommand() *adapters.CatalogBuilder {
	builder := adapters.NewBuilder("Bench API", "1.0.0")
	builder.AddService(
		"svc", "Service", "1.0.0", "",
		catalog.Command[benchCreateUser]("user.create"),
	)

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
			command.CatalogMeta{ //nolint:staticcheck
				Name: fmt.Sprintf("Cmd%d", i), Version: "1.0.0",
			},
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

	for b.Loop() {
		exp := eventcatalog.NewExporter(b.TempDir())
		_ = exp.Export(cat)
	}
}
