package adapters

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	d2exporter "github.com/larsartmann/go-cqrs-lite/catalog/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

// CatalogBuilder accumulates services and their messages, then builds a catalog
// with export capabilities. It wraps a catalog.Registry for accumulation and
// adds convenience methods for export formats.
type CatalogBuilder struct {
	registry *catalog.Registry
}

// NewBuilder creates a new catalog builder with the given title and version.
func NewBuilder(title, version string) *CatalogBuilder {
	return &CatalogBuilder{
		registry: catalog.NewRegistry(title, version),
	}
}

// Build creates the final immutable catalog from all registered services and domains.
func (b *CatalogBuilder) Build() *catalog.Catalog {
	return b.registry.Build()
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

// ExportD2 creates a D2 diagram string from the catalog.
func (b *CatalogBuilder) ExportD2(
	title, version string,
	opts ...d2exporter.Option,
) string {
	cat := b.Build()
	exp := d2exporter.NewExporter(title, version, opts...)

	return exp.Export(cat)
}

func (b *CatalogBuilder) addMessageToService(
	serviceID string,
	kind catalog.MessageKind,
	msg catalog.Message,
) {
	switch kind {
	case catalog.CommandMessage:
		b.registry.AddCommand(serviceID, msg)
	case catalog.EventMessage:
		b.registry.AddEvent(serviceID, msg)
	case catalog.QueryMessage:
		b.registry.AddQuery(serviceID, msg)
	}
}

// AddService registers a service with optional summary.
func (b *CatalogBuilder) AddService(id, name, version, summary string) {
	b.registry.SetServiceMeta(id, name, version, summary)
}

// AddDomain registers a domain and associates it with services.
func (b *CatalogBuilder) AddDomain(id, name, summary string, serviceIDs []string) {
	b.registry.AddDomain(catalog.Domain{
		ID:       id,
		Name:     name,
		Version:  "1.0.0",
		Summary:  summary,
		Services: serviceIDs,
	})
}

// AddServiceToDomain associates an existing service with a domain.
func (b *CatalogBuilder) AddServiceToDomain(serviceID, domainID string) error {
	err := b.registry.AddServiceToDomain(serviceID, domainID)
	if err != nil {
		return fmt.Errorf("add service %q to domain %q: %w", serviceID, domainID, err)
	}

	return nil
}

// AddChannel registers a channel in the catalog.
func (b *CatalogBuilder) AddChannel(ch catalog.Channel) {
	b.registry.AddChannel(ch)
}
