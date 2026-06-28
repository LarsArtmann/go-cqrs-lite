package docserver

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/eventcatalog"
)

// GenerateEventCatalog writes EventCatalog MDX files to the given outputDir.
// Call this at application startup to generate documentation files that
// can be served by the EventCatalog CLI or a static file server.
//
// This is a build-time/startup-time operation, not an HTTP handler.
// EventCatalog expects a directory of MDX files served statically — there
// is no meaningful way to serve it as a single HTTP response.
//
// The returned error is already classified as Infrastructure via go-error-family.
func GenerateEventCatalog(cat *catalog.Catalog, outputDir string) error {
	exporter := eventcatalog.NewExporter(outputDir)

	return exporter.Export(cat)
}
