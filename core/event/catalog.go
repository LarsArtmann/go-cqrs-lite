package event

// CatalogMeta contains documentation metadata for auto-catalog generation.
//
// Deprecated: The zero-cost catalog API in github.com/larsartmann/go-cqrs-lite/catalog
// auto-derives all metadata from Go struct types. Use catalog.Command[T](),
// catalog.Event[T](), or catalog.Query[T]() instead of implementing Catalogable.
type CatalogMeta struct {
	Name          string
	Version       string
	Summary       string
	AggregateType AggregateType
}
