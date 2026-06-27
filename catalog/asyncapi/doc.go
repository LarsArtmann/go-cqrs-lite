// Package asyncapi generates AsyncAPI 3.0 documents from a [catalog.Catalog].
//
// Each service's commands become receive operations, events become send/receive
// operations (based on direction), and queries become request/reply operations.
// The resulting Document maps services, channels, messages, and component
// schemas into the AsyncAPI 3.0 structure.
//
//	exp := asyncapi.NewExporter("order-service", "1.0.0")
//	doc := exp.Export(cat)
//	yaml, _ := doc.MarshalYAML()
package asyncapi
