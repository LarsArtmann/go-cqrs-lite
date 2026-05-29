package catalog

// Option applies configuration to an exporter of type T.
type Option[T any] func(*T)

// Exporter converts a Catalog into an output format.
// Each exporter sub-package (asyncapi, d2, eventcatalog, openapi)
// provides a concrete implementation.
type Exporter[T any] interface {
	Export(cat *Catalog) T
}

// ErrorExporter is a variant of Exporter that can fail.
// Use this for exporters that write to filesystem or external services.
type ErrorExporter interface {
	Export(cat *Catalog) error
}
