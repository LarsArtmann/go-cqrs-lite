package catalog

// Option applies configuration to an exporter of type T.
type Option[T any] func(*T)

// Exporter converts a Catalog into an output format.
// Each exporter sub-package (asyncapi, d2, eventcatalog, openapi)
// provides a concrete implementation.
type Exporter[T any] interface {
	Export(cat *Catalog) T
}
