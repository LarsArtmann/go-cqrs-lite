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
