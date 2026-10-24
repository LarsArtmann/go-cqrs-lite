package architecture_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type benchPayload struct {
	ID      string `json:"id"      doc:"Identifier"`
	Counter int    `json:"counter" doc:"A counter"`
}

// largeCatalog generates services × messagesPerService resources: every
// service carries commands, events and queries, belongs to a domain and
// reads a data store — the shape a real federation hub export has.
func largeCatalog(services, messagesPerService int) *catalog.Catalog {
	builder := catalog.NewBuilder("Bench", "1.0.0")

	for i := range services {
		id := catalog.ServiceID(fmt.Sprintf("svc-%04d", i))

		msgs := make([]catalog.MessageConfig, 0, messagesPerService)

		for j := range messagesPerService {
			mid := catalog.MessageID(fmt.Sprintf("%s-msg-%02d", id, j))

			switch j % 3 {
			case 0:
				msgs = append(msgs, catalog.Command[benchPayload](mid))
			case 1:
				msgs = append(msgs, catalog.Event[benchPayload](mid, catalog.Sends))
			default:
				msgs = append(msgs, catalog.Query[benchPayload](mid))
			}
		}

		builder.AddService(
			id, "Service "+string(id), "1.0.0",
			"Benchmark service "+string(id)+".", msgs...,
		)
		builder.ConfigureService(
			id,
			catalog.ServiceReadsFrom(catalog.DataStoreID(fmt.Sprintf("store-%04d", i/10))),
		)
		builder.AddDomain(
			catalog.DomainID(fmt.Sprintf("dom-%04d", i/10)),
			"Domain "+fmt.Sprintf("%04d", i/10),
			"1.0.0", "Benchmark domain.", id,
		)
	}

	for i := range services / 10 {
		builder.AddDataStore(catalog.DataStore{
			ID:         catalog.DataStoreID(fmt.Sprintf("store-%04d", i)),
			Name:       catalog.Name("Store " + fmt.Sprintf("%04d", i)),
			Version:    "1.0.0",
			Summary:    "Benchmark store.",
			Technology: "postgres@16",
		})
	}

	return builder.Build()
}

// BenchmarkExportLargeCatalog measures the whole-pipeline render of a
// 1 000-service / 5 000-message catalog. Measured 2026-10-17 on a quiet
// host: ~175 ms per render (warm caches; cold first render ≈ 615 ms) and
// 15.65 MB of HTML — a one-shot cost at build or deploy time, not per
// request. The size is why sections carry `content-visibility: auto`: the
// browser only lays out the viewport, so the file stays scrollable while
// the rest stays unpainted.
func BenchmarkExportLargeCatalog(b *testing.B) {
	cat := largeCatalog(1_000, 5)

	exp := architecture.NewExporter(
		architecture.WithTitle("Bench"),
		architecture.WithCommand("bench"),
	)

	var size int

	b.ResetTimer()

	for range b.N {
		out, err := exp.Export(cat)
		if err != nil {
			b.Fatalf("Export: %v", err)
		}

		size = len(out)
	}

	b.ReportMetric(float64(size)/1e6, "MB/doc")
}

// BenchmarkInspectLargeCatalog measures the dry-run report of the same
// 1 000-service / 5 000-message catalog — the cost a hub pays per source in
// CI before rendering (ADR-0160 `-check`). Measured 2026-10-24: ~2–3 ms per
// Inspect warm (32-core host shared with concurrent sessions; the plan +
// collect + report path, no render) — two orders of magnitude under the
// ~175 ms render it gates.
func BenchmarkInspectLargeCatalog(b *testing.B) {
	cat := largeCatalog(1_000, 5)

	exp := architecture.NewExporter(
		architecture.WithTitle("Bench"),
		architecture.WithCommand("bench"),
		architecture.WithInclude(architecture.Selector{Kind: "domain", ID: "dom-0000"}),
	)

	b.ResetTimer()

	for range b.N {
		report := exp.Inspect(cat)
		if report.Failed() {
			b.Fatalf("Inspect reported failure: %+v", report)
		}
	}
}
