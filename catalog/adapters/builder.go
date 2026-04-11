package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

// CatalogBuilder accumulates services and their messages, then builds a catalog.
type CatalogBuilder struct {
	title    string
	version  string
	services map[string]catalog.Service
	domains  map[string]catalog.Domain
}

// NewBuilder creates a new catalog builder with the given title and version.
func NewBuilder(title, version string) *CatalogBuilder {
	return &CatalogBuilder{
		title:    title,
		version:  version,
		services: make(map[string]catalog.Service),
		domains:  make(map[string]catalog.Domain),
	}
}

// Build creates the final immutable catalog from all registered services and domains.
func (b *CatalogBuilder) Build() *catalog.Catalog {
	services := make([]catalog.Service, 0, len(b.services))
	for _, svc := range b.services {
		services = append(services, svc)
	}

	domains := make([]catalog.Domain, 0, len(b.domains))
	for _, d := range b.domains {
		domains = append(domains, d)
	}

	return &catalog.Catalog{
		Title:    b.title,
		Version:  b.version,
		Services: services,
		Domains:  domains,
	}
}

// ExportEventCatalog writes the catalog to disk in EventCatalog format.
func (b *CatalogBuilder) ExportEventCatalog(outputDir string) error {
	cat := b.Build()
	exp := eventcatalog.NewExporter(outputDir)

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

func (b *CatalogBuilder) addMessageToService(
	serviceID string,
	kind catalog.MessageKind,
	msg catalog.Message,
) {
	svc, ok := b.services[serviceID]
	if !ok {
		svc = catalog.Service{ID: serviceID, Name: serviceID}
	}

	switch kind {
	case catalog.CommandMessage:
		svc.Commands = append(svc.Commands, msg)
	case catalog.EventMessage:
		svc.Events = append(svc.Events, msg)
	case catalog.QueryMessage:
		svc.Queries = append(svc.Queries, msg)
	}

	b.services[serviceID] = svc
}

// ensureService creates the service entry if it doesn't exist.
func (b *CatalogBuilder) ensureService(id, name string) {
	if _, ok := b.services[id]; !ok {
		b.services[id] = catalog.Service{ID: id, Name: name}
	}
}

// AddService registers a service with optional summary.
func (b *CatalogBuilder) AddService(id, name, version, summary string) {
	b.ensureService(id, name)
	svc := b.services[id]
	svc.Version = version
	b.services[id] = svc

	if summary != "" {
		svc.Summary = summary
		b.services[id] = svc
	}
}

// AddDomain registers a domain and associates it with services.
func (b *CatalogBuilder) AddDomain(id, name, summary string, serviceIDs []string) {
	b.domains[id] = catalog.Domain{
		ID:       id,
		Name:     name,
		Version:  "1.0.0",
		Summary:  summary,
		Services: serviceIDs,
	}
}

// AddServiceToDomain associates an existing service with a domain.
func (b *CatalogBuilder) AddServiceToDomain(serviceID, domainID string) {
	d, ok := b.domains[domainID]
	if !ok {
		return
	}

	d.Services = append(d.Services, serviceID)
	b.domains[domainID] = d
}
