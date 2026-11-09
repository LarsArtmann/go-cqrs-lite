package docserver

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"
)

func (ds *DocsServer) buildOpenAPI() *openapi.Document {
	cat := ds.provider()

	var opts []openapi.Option

	if ds.config.Description != "" {
		opts = append(opts, openapi.WithDescription(ds.config.Description))
	}

	if ds.config.BasePath != "" {
		opts = append(opts, openapi.WithBasePath(ds.config.BasePath))
	}

	return openapi.NewExporter(ds.config.ServiceName, ds.config.Version, opts...).Export(cat)
}

// buildOpenAPIForRequest pins the OpenAPI servers entry to the scheme+host of
// the current request. The exporter emits a relative "." server URL, which
// resolves against the DOCUMENT location (e.g. /api/docs/) — sending the
// Scalar Try-It console to /api/docs/api/... instead of /api/... Deriving the
// base from the request (honoring X-Forwarded-Proto behind reverse proxies)
// makes interactive testing work on any deployment host.
func (ds *DocsServer) buildOpenAPIForRequest(r *http.Request) *openapi.Document {
	doc := ds.buildOpenAPI()

	if base := requestBaseURL(r); base != "" {
		doc.Servers = []openapi.Server{{URL: base, Description: "Current server"}}
	}

	return doc
}

// requestBaseURL returns scheme://host for the request, or "" when the
// request carries no Host header.
func requestBaseURL(r *http.Request) string {
	if r.Host == "" {
		return ""
	}

	scheme := "http"

	switch {
	case r.Header.Get("X-Forwarded-Proto") != "":
		scheme = r.Header.Get("X-Forwarded-Proto")
	case r.TLS != nil:
		scheme = "https"
	}

	return scheme + "://" + r.Host
}

func (ds *DocsServer) buildAsyncAPI() *asyncapi.Document {
	cat := ds.provider()

	opts := []asyncapi.Option{
		asyncapi.WithServer(
			ds.config.AsyncAPIServer.Name,
			ds.config.AsyncAPIServer.Host,
			ds.config.AsyncAPIServer.Protocol,
		),
	}

	if ds.config.Description != "" {
		opts = append(opts, asyncapi.WithDescription(ds.config.Description))
	}

	return asyncapi.NewExporter(ds.config.ServiceName, ds.config.Version, opts...).Export(cat)
}

func (ds *DocsServer) buildCatalog() *catalog.Catalog {
	return ds.provider()
}
