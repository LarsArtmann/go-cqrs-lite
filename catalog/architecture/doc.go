// Package architecture renders a [catalog.Catalog] as ONE self-contained HTML
// architecture reference.
//
// The output mirrors the "architecture reference" from EventCatalog 5.1: a
// single .html file with an overview, a filterable sidebar listing every
// resource, a section per resource (summary, badges, relationships as links,
// message attributes, example payloads, flow steps), full-text search
// (Ctrl/Cmd-K) and light/dark themes. Styles, scripts, schemas and examples are
// inlined, so the file needs no server and no Node toolchain: attach it to an
// email, keep it with an audit, or serve it from any static host.
//
//	exp := architecture.NewExporter(architecture.WithTitle("Acme Inc"))
//	html, err := exp.Export(cat)              // []byte, self-contained
//	err = exp.ExportFile(cat, "architecture.html")
//
// A consumer that already serves a documentation site can embed the rendered
// bytes with //go:embed and hand them out from an existing route instead.
//
// Unlike [github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog], which
// writes an MDX tree for the Node EventCatalog CLI, this package renders the
// final HTML itself and links to nothing outside the file. It uses only the
// standard library: no templ, no EventCatalog.
package architecture
