package docserver

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/openapi"
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
