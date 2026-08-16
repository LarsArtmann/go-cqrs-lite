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

// indexPageData carries everything the documentation index page renders.
type indexPageData struct {
	Brand        string
	CatalogTitle string
	Version      string
	Description  string
	DocsPath     string
	Stats        catalogStats
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
		services = append(services, serviceCard{
			ID:       string(svc.ID),
			Name:     cmp.Or(string(svc.Name), string(svc.ID)),
			Version:  string(svc.Version),
			Summary:  string(svc.Summary),
			Commands: countUniqueMessages(seenCommands, svc.Commands),
			Events:   countUniqueMessages(seenEvents, svc.Events),
			Queries:  countUniqueMessages(seenQueries, svc.Queries),
		})
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
		Services: services,
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
