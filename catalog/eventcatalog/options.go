package eventcatalog

// Option configures an Exporter. Options are applied in order by
// NewExporter; later options win.
type Option func(*Exporter)

// WithSkipBootstrapFiles omits the generated bootstrap files —
// `eventcatalog.config.js` and `package.json` — from the export.
//
// Federation hubs that union-merge per-service exports keep their OWN
// bootstrap files (site title, pinned core version, dependencies): a merged
// tree carrying per-source package.json/config files would fight the hub's,
// and the merge would depend on fragile first-wins copy semantics. CI export
// commands for such hubs pass this option; local dev exports keep the
// default so the output directory remains directly buildable
// (`npm install && npx eventcatalog build`).
func WithSkipBootstrapFiles() Option {
	return func(e *Exporter) {
		e.skipBootstrapFiles = true
	}
}

// WithPlainRefIDs emits resource references as bare IDs instead of the
// default composite "<id>-<version>" Astro entry IDs — producers/consumers
// become plain service IDs and channel message pointers carry the plain
// message ID (the version stays a separate field).
//
// Use for GOVERNANCE exports consumed by tooling that keys resources by
// frontmatter ID: `@eventcatalog/linter` indexes every collection by the
// frontmatter `id` and can never resolve the composite form, so
// `refs/resource-exists` errors on every ref in a default export. A
// plain-ref export lints clean.
//
// Do NOT use for trees that `@eventcatalog/core` builds: its Astro content
// layer keys entries by "<id>-<version>" and logs `Invalid content
// reference` for bare IDs (the visualiser graph edges use the same keys).
// That is why composite is the default and this is an option — one catalog,
// two exports: render the default tree, lint the plain one.
func WithPlainRefIDs() Option {
	return func(e *Exporter) {
		e.plainRefIDs = true
	}
}
