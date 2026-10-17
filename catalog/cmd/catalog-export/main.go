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
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
)

// selectorList collects repeatable string flags (-include kind:id …).
type selectorList []string

func (l *selectorList) String() string {
	return strings.Join(*l, ", ")
}

func (l *selectorList) Set(value string) error {
	*l = append(*l, value)

	return nil
}

var errBadSelector = errors.New("selector must be kind:id")

// parseSelectors turns "kind:id" flags into selectors; a malformed flag is
// a usage error, not a silent skip.
func parseSelectors(flags selectorList) ([]architecture.Selector, error) {
	selectors := make([]architecture.Selector, 0, len(flags))

	for _, raw := range flags {
		kind, id, found := strings.Cut(raw, ":")
		if !found || kind == "" || id == "" {
			return nil, fmt.Errorf(
				"catalog-export: selector %q is not kind:id: %w",
				raw,
				errBadSelector,
			)
		}

		selectors = append(selectors, architecture.Selector{Kind: kind, ID: id})
	}

	return selectors, nil
}

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
	reference := flag.String(
		"architecture",
		"",
		"also write a self-contained architecture reference HTML file to this path",
	)

	var (
		includes    selectorList
		excludes    selectorList
		neverExport selectorList
	)

	flag.Var(
		&includes,
		"include",
		"keep only the selected resources and their closure, kind:id (repeatable)",
	)
	flag.Var(
		&excludes,
		"exclude",
		"drop the selected resources from the export, kind:id (repeatable)",
	)
	flag.Var(
		&neverExport,
		"never-export",
		"redact the selected resources from any export, kind:id (repeatable)",
	)

	references := flag.String(
		"references",
		"",
		"how out-of-scope references render: omit (default), name-only, include",
	)
	hide := flag.String(
		"hide",
		"",
		"comma-separated reference parts to suppress: owners,repository,examples,specifications,docs",
	)
	check := flag.Bool(
		"check",
		false,
		"print the export check report instead of writing files; exits 1 when a selector matches nothing",
	)
	spec := flag.String(
		"spec",
		"",
		"load the export policy from a markdown file with YAML frontmatter (title and include required; body becomes the introduction)",
	)

	flag.Parse()

	cat := buildCatalog()

	var specOptions []architecture.Option

	if *spec != "" {
		loaded, err := architecture.LoadExport(*spec)
		if err != nil {
			log.Fatalf("catalog-export: %v", err)
		}

		specOptions = loaded.Options()
	}

	includeSelectors, err := parseSelectors(includes)
	if err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	excludeSelectors, err := parseSelectors(excludes)
	if err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	neverExportSelectors, err := parseSelectors(neverExport)
	if err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	options := append([]architecture.Option{
		architecture.WithCommand("catalog-export -architecture " + *reference),
	}, specOptions...)

	if len(includeSelectors) > 0 {
		options = append(options, architecture.WithInclude(includeSelectors...))
	}

	if len(excludeSelectors) > 0 {
		options = append(options, architecture.WithExclude(excludeSelectors...))
	}

	if len(neverExportSelectors) > 0 {
		options = append(options, architecture.WithNeverExport(neverExportSelectors...))
	}

	if *references != "" {
		options = append(
			options,
			architecture.WithReferences(architecture.ReferencesMode(*references)),
		)
	}

	if *hide != "" {
		parts := strings.Split(*hide, ",")

		sections := make([]architecture.HideSection, 0, len(parts))
		for _, part := range parts {
			sections = append(sections, architecture.HideSection(strings.TrimSpace(part)))
		}

		options = append(options, architecture.WithHide(sections...))
	}

	exp := architecture.NewExporter(options...)

	if *check {
		report := exp.Inspect(cat)
		fmt.Fprint(os.Stdout, report.String())

		if report.Failed() {
			os.Exit(1)
		}

		return
	}

	if err := eventcatalog.NewExporter(*output).Export(cat); err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	fmt.Fprintf(os.Stdout, "catalog-export: wrote EventCatalog tree to %s\n", *output)

	if *reference == "" {
		return
	}

	if err := exp.ExportFile(cat, *reference); err != nil {
		log.Fatalf("catalog-export: %v", err)
	}

	fmt.Fprintf(os.Stdout, "catalog-export: wrote architecture reference to %s\n", *reference)
}
