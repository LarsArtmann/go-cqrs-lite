// Package eventcatalog writes an EventCatalog-compatible file tree from a
// [catalog.Catalog].
//
// It generates MDX pages with YAML frontmatter, JSON schemas, an llms.txt index,
// and the eventcatalog.config.js / package.json needed to serve the catalog.
// Producers and consumers are auto-derived before writing services, messages,
// domains, channels, data stores, flows, teams, and users.
//
//	exp := eventcatalog.NewExporter("./eventcatalog")
//	_ = exp.Export(cat) // writes the full output tree to disk
package eventcatalog
