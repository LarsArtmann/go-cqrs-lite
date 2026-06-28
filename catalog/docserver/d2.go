package docserver

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/d2"
)

// D2Option configures the D2 diagram handler.
type D2Option func(*d2Config)

type d2Config struct {
	description string
}

// WithD2Description sets the description on the generated D2 diagram.
func WithD2Description(desc string) D2Option {
	return func(c *d2Config) {
		c.description = desc
	}
}

// D2Handler returns an http.HandlerFunc that serves the catalog
// as a D2 architecture diagram (text/plain).
//
// This is a standalone handler — it does not require a DocsServer.
// Use it when you only need the D2 diagram endpoint, or alongside
// a DocsServer for the full documentation suite.
func D2Handler(cat *catalog.Catalog, opts ...D2Option) http.HandlerFunc {
	cfg := d2Config{} //nolint:exhaustruct // optional fields

	for _, opt := range opts {
		opt(&cfg)
	}

	exporterOpts := []d2.Option{}
	if cfg.description != "" {
		exporterOpts = append(exporterOpts, d2.WithDescription(cfg.description))
	}

	exporter := d2.NewExporter(
		string(cat.Title),
		string(cat.Version),
		exporterOpts...,
	)

	return func(w http.ResponseWriter, _ *http.Request) {
		text := exporter.Export(cat)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(text))
	}
}
