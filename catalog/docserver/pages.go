package docserver

import (
	"cmp"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// catalogStats aggregates deduplicated message counts for the docs index page.
// Messages referenced by several services are counted once.
type catalogStats struct {
	Services   int
	Commands   int
	Events     int
	Queries    int
	Channels   int
	DataStores int
}

// serviceCard is the index-page summary of one catalog service.
type serviceCard struct {
	ID       string
	Name     string
	Version  string
	Summary  string
	Commands int
	Events   int
	Queries  int
}

// rawLink is a secondary raw-artifact link (JSON/YAML/D2 text) on a spec card.
type rawLink struct {
	Href  string
	Label string
}

// specLink is one documentation artifact card on the index page.
type specLink struct {
	Title    string
	Subtitle string
	PageHref string
	Raw      []rawLink
}

// indexPageData carries everything the documentation index page renders.
type indexPageData struct {
	Brand        string
	CatalogTitle string
	Version      string
	Description  string
	DocsPath     string
	Stats        catalogStats
	Specs        []specLink
	Services     []serviceCard
}

// newIndexPageData derives the index-page view model from server config and
// the current catalog snapshot.
func newIndexPageData(cfg Config, cat *catalog.Catalog) indexPageData {
	seenCommands := map[catalog.MessageID]bool{}
	seenEvents := map[catalog.MessageID]bool{}
	seenQueries := map[catalog.MessageID]bool{}

	services := make([]serviceCard, 0, len(cat.Services))
	for _, svc := range cat.Services {
		// Per-service counts use local sets so messages shared across
		// services still appear on every service card; the global sets
		// below drive the deduplicated catalog totals.
		localCommands := map[catalog.MessageID]bool{}
		localEvents := map[catalog.MessageID]bool{}
		localQueries := map[catalog.MessageID]bool{}

		services = append(services, serviceCard{
			ID:       string(svc.ID),
			Name:     cmp.Or(string(svc.Name), string(svc.ID)),
			Version:  string(svc.Version),
			Summary:  string(svc.Summary),
			Commands: countUniqueMessages(localCommands, svc.Commands),
			Events:   countUniqueMessages(localEvents, svc.Events),
			Queries:  countUniqueMessages(localQueries, svc.Queries),
		})

		countUniqueMessages(seenCommands, svc.Commands)
		countUniqueMessages(seenEvents, svc.Events)
		countUniqueMessages(seenQueries, svc.Queries)
	}

	return indexPageData{
		Brand:        cmp.Or(cfg.ServiceName, string(cat.Title)),
		CatalogTitle: cmp.Or(string(cat.Title), cfg.ServiceName),
		Version:      cmp.Or(string(cat.Version), cfg.Version),
		Description:  cfg.Description,
		DocsPath:     cfg.DocsPath,
		Stats: catalogStats{
			Services:   len(cat.Services),
			Commands:   len(seenCommands),
			Events:     len(seenEvents),
			Queries:    len(seenQueries),
			Channels:   len(cat.Channels),
			DataStores: len(cat.DataStores),
		},
		Specs:    indexSpecLinks(cfg.DocsPath),
		Services: services,
	}
}

// indexSpecLinks builds the documentation artifact cards for the index page.
func indexSpecLinks(docsPath string) []specLink {
	return []specLink{
		{
			Title:    "OpenAPI reference",
			Subtitle: "Interactive REST API documentation (Scalar)",
			PageHref: docsPath + "/openapi",
			Raw: []rawLink{
				{Href: docsPath + "/openapi.json", Label: "JSON"},
				{Href: docsPath + "/openapi.yaml", Label: "YAML"},
			},
		},
		{
			Title:    "AsyncAPI reference",
			Subtitle: "Interactive event documentation (AsyncAPI React)",
			PageHref: docsPath + "/asyncapi",
			Raw: []rawLink{
				{Href: docsPath + "/asyncapi.json", Label: "JSON"},
				{Href: docsPath + "/asyncapi.yaml", Label: "YAML"},
			},
		},
		{
			Title:    "Architecture diagram",
			Subtitle: "D2 diagram source generated from the catalog",
			PageHref: docsPath + "/d2",
			Raw: []rawLink{
				{Href: docsPath + "/d2.txt", Label: "D2 text"},
			},
		},
		{
			Title:    "Catalog JSON",
			Subtitle: "The raw catalog snapshot this documentation is built from",
			PageHref: docsPath + "/catalog.json",
		},
	}
}

// countUniqueMessages counts messages whose key has not been seen yet,
// mutating the seen set.
func countUniqueMessages(seen map[catalog.MessageID]bool, msgs []catalog.Message) int {
	added := 0

	for _, msg := range msgs {
		if key := catalog.Key(msg); !seen[key] {
			seen[key] = true
			added++
		}
	}

	return added
}
