// Package d2 generates D2 diagrams from a [catalog.Catalog].
//
// It renders each service as a container with its command, query, and event
// nodes, draws internal edges (receives/publishes/handles), cross-service event
// edges (matching publishers to subscribers), and domain groupings. The result
// is a D2 DSL string ready to render with the D2 CLI or any D2-compatible tool.
//
//	exp := d2.NewExporter("Orders", "1.0.0")
//	diagram := exp.Export(cat) // D2 DSL text
package d2
