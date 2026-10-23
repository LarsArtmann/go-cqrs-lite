package docserver

import (
	"net/http"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// architectureCache caches one rendered reference keyed on the catalog
// instance. Catalogs are immutable after Build, so the same instance always
// renders the same bytes; a provider that rebuilds the catalog per call
// pays a re-render per request, which is the honest price of freshness.
type architectureCache struct {
	mu     sync.Mutex
	cat    *catalog.Catalog
	render []byte
	err    error
	loaded bool
}

// load returns the cached render for cat, rendering once when the catalog
// instance changed.
func (c *architectureCache) load(
	cat *catalog.Catalog,
	render func(*catalog.Catalog) ([]byte, error),
) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// loaded also caches a failed render: retrying a deterministic failure on
	// every request would turn one render error into a per-request cost.
	if c.loaded && c.cat == cat {
		return c.render, c.err
	}

	c.cat = cat
	c.loaded = true
	c.render, c.err = render(cat)

	return c.render, c.err
}

// ArchitectureReferenceHandler serves the self-contained architecture
// reference as HTML. The artifact carries its own inline styles and
// scripts, so the docserver CSP policy (which nonces templ-page bundles)
// is deliberately not applied to it.
func (ds *DocsServer) ArchitectureReferenceHandler() http.HandlerFunc {
	return ds.serveArchitecture
}

func (ds *DocsServer) serveArchitecture(w http.ResponseWriter, _ *http.Request) {
	render, err := ds.architecture.load(ds.provider(), func(cat *catalog.Catalog) ([]byte, error) {
		exp := architecture.NewExporter(
			architecture.WithTitle(ds.config.ServiceName),
			architecture.WithDescription(ds.config.Description),
			architecture.WithCommand("docserver "+ds.config.DocsPath+"/architecture"),
		)

		return exp.Export(cat)
	})
	if err != nil {
		http.Error(
			w,
			"architecture reference render failed: "+err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(render)
}
