// Command catalog-export is a COPY-PASTE TEMPLATE for feeding the
// architecture federation hub (eventcatalog-hub CI → catalog.home.lan).
//
// Drop this file into YOUR service repo as cmd/catalog-export/main.go and
// replace buildCatalog() with your own message registrations — everything
// else stays. The hub CI clones your repo and runs the command headlessly
// (see the hub's sources.json `build`/`export` fields):
//
//	go build -o work/bin/catalog-export ./cmd/catalog-export
//	work/bin/catalog-export -o work/out/<your-service>
//
// The command must: exit 0 without serving anything, and write a complete
// EventCatalog MDX tree (services/, events/, commands/, queries/, …) under
// the output dir. Producer/consumer links across services are union-merged
// by the hub, so register directions (Sends/Receives) faithfully.
//
// Alternative shapes that already work: a `catalog` subcommand on your
// service CLI (bank-sync pattern: `bank-sync catalog --format eventcatalog
// -o work/out/bank-sync`) or a headless flag on a docs demo (cqrs-htmx
// pattern: `catalog-demo -eventcatalog <dir> -export-only`).
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
)

// Replace this demo registration with your own types — the rest of the
// file is repo-agnostic.
func buildCatalog() *catalog.Catalog {
	b := simple.New("Demo Service", "1.0.0",
		simple.WithServiceSummary("Replace with your service summary"),
	)

	simple.Command[demoCommand](b, "demo-command",
		catalog.WithSummary("A demo command"),
	)
	simple.Event[demoEvent](b, "demo.event", catalog.Sends,
		catalog.WithSummary("A demo event"),
	)

	return b.Build()
}

type demoCommand struct {
	ID string `json:"id" doc:"Demo identifier" example:"demo_123"`
}

type demoEvent struct {
	ID string `json:"id" doc:"Demo identifier" example:"demo_123"`
}

func main() {
	output := flag.String(
		"o",
		"work/out/catalog-export",
		"output directory for the EventCatalog MDX tree",
	)

	flag.Parse()

	if err := eventcatalog.NewExporter(*output).Export(buildCatalog()); err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	fmt.Printf("catalog-export: wrote EventCatalog tree to %s\n", *output)
}
