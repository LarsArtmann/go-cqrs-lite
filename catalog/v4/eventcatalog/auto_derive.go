package eventcatalog

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

// autoDeriveProducersConsumers returns a copy of the catalog with
// producer/consumer relationships auto-derived from service sends/receives.
// The derivation itself is exported on the catalog type
// (Catalog.DeriveProducersConsumers) so validation gates render the same
// enriched graph the exporter does.
func autoDeriveProducersConsumers(cat *catalog.Catalog) *catalog.Catalog {
	return cat.DeriveProducersConsumers()
}
