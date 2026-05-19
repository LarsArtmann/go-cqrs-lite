package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	d2exporter "github.com/larsartmann/go-cqrs-lite/catalog/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/openapi"
)

// CatalogBuilder accumulates services and their messages, then builds a catalog
// with export capabilities. It wraps catalog.Builder for accumulation and
// adds convenience methods for export formats.
//
// Deprecated: Use catalog.Builder directly with catalog.Command[T](),
// catalog.Event[T](), and catalog.Query[T]() for the zero-cost catalog API.
// This type is retained for backward compatibility only.
type CatalogBuilder struct {
	builder *catalog.Builder
}

// NewBuilder creates a new catalog builder with the given title and version.
func NewBuilder(title, version string) *CatalogBuilder {
	return &CatalogBuilder{
		builder: catalog.NewBuilder(title, version),
	}
}

// Build creates the final immutable catalog from all registered services and domains.
func (b *CatalogBuilder) Build() *catalog.Catalog {
	return b.builder.Build()
}

// ExportEventCatalog writes the catalog to disk in EventCatalog format.
func (b *CatalogBuilder) ExportEventCatalog(outputDir string) error {
	cat := b.Build()
	exp := eventcatalog.NewExporter(outputDir)

	//nolint:wrapcheck // Export is called in a wrapper; caller handles error
	return exp.Export(cat)
}

// ExportAsyncAPI creates an AsyncAPI 3.0 document from the catalog.
func (b *CatalogBuilder) ExportAsyncAPI(
	serviceName, version string,
	opts ...asyncapi.Option,
) (*asyncapi.Document, error) {
	cat := b.Build()
	exp := asyncapi.NewExporter(serviceName, version, opts...)

	return exp.Export(cat), nil
}

// ExportOpenAPI creates an OpenAPI 3.0 document from the catalog.
func (b *CatalogBuilder) ExportOpenAPI(
	title, version string,
	opts ...openapi.Option,
) *openapi.Document {
	cat := b.Build()
	exp := openapi.NewExporter(title, version, opts...)

	return exp.Export(cat)
}

// ExportD2 creates a D2 diagram string from the catalog.
func (b *CatalogBuilder) ExportD2(
	title, version string,
	opts ...d2exporter.Option,
) string {
	cat := b.Build()
	exp := d2exporter.NewExporter(title, version, opts...)

	return exp.Export(cat)
}

// AddService registers a service with messages.
// Messages can be created with catalog.Command[T](), catalog.Event[T](),
// and catalog.Query[T]().
func (b *CatalogBuilder) AddService(
	id, name, version, summary string,
	msgs ...catalog.MessageConfig,
) {
	b.builder.AddService(id, name, version, summary, msgs...)
}

// AddDomain registers a domain and associates it with services.
func (b *CatalogBuilder) AddDomain(id, name, summary string, serviceIDs []string) {
	b.builder.AddDomain(id, name, "1.0.0", summary, serviceIDs...)
}

// AddServiceToDomain associates an existing service with a domain.
func (b *CatalogBuilder) AddServiceToDomain(serviceID, domainID string) error {
	return b.builder.Registry().AddServiceToDomain(serviceID, domainID)
}

// AddChannel registers a channel in the catalog.
func (b *CatalogBuilder) AddChannel(ch catalog.Channel) {
	// Channels are not yet supported by catalog.Builder.
	// This method is retained for backward compatibility.
}

// AddCommand adds a command message directly to a service.
// Deprecated: Use AddService with catalog.Command[T]() instead.
func (b *CatalogBuilder) AddCommand(serviceID string, msg catalog.Message) {
	b.builder.Registry().AddCommand(serviceID, msg)
}

// AddEvent adds an event message directly to a service.
// Deprecated: Use AddService with catalog.Event[T]() instead.
func (b *CatalogBuilder) AddEvent(serviceID string, msg catalog.Message) {
	b.builder.Registry().AddEvent(serviceID, msg)
}

// AddQuery adds a query message directly to a service.
// Deprecated: Use AddService with catalog.Query[T]() instead.
func (b *CatalogBuilder) AddQuery(serviceID string, msg catalog.Message) {
	b.builder.Registry().AddQuery(serviceID, msg)
}
