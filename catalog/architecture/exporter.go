package architecture

import (
	"io"
	"os"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// outputPerm is the file mode used by [Exporter.ExportFile].
const outputPerm = 0o600

// config is the resolved exporter configuration.
type config struct {
	title       string
	description string
	command     string
	logo        string
	generatedAt time.Time
	sel         *Selection
	references  ReferencesMode
	hide        visibility
	introduction string
}

// Exporter renders a catalog as one self-contained HTML architecture
// reference. The zero value is not usable; create one with [NewExporter].
type Exporter struct {
	cfg config
}

// Option configures an [Exporter].
type Option func(*Exporter)

// WithTitle overrides the document title and heading. It defaults to the
// catalog's own title.
func WithTitle(title string) Option {
	return func(e *Exporter) { e.cfg.title = title }
}

// WithDescription sets the sentence shown under the title.
func WithDescription(description string) Option {
	return func(e *Exporter) { e.cfg.description = description }
}

// WithCommand records the command that regenerates the document. It is shown
// on the overview so a reader can reproduce the file.
func WithCommand(command string) Option {
	return func(e *Exporter) { e.cfg.command = command }
}

// WithLogo sets the brand image shown in the top bar. Only http(s) and data
// URLs are rendered; anything else is ignored.
func WithLogo(logo string) Option {
	return func(e *Exporter) { e.cfg.logo = logo }
}

// WithGeneratedAt stamps the document with a fixed time. It defaults to the
// current time; set it for reproducible output.
func WithGeneratedAt(at time.Time) Option {
	return func(e *Exporter) { e.cfg.generatedAt = at.UTC() }
}

// WithInclude restricts the export to the selected resources and their
// natural closure (ADR-0160). An empty include keeps the whole catalog.
func WithInclude(selectors ...Selector) Option {
	return func(e *Exporter) {
		e.cfg.sel = e.cfg.sel.including(selectors...)
	}
}

// WithExclude drops the selected resources from the export even when an
// include would keep them; excludes always win (ADR-0160).
func WithExclude(selectors ...Selector) Option {
	return func(e *Exporter) {
		e.cfg.sel = e.cfg.sel.excluding(selectors...)
	}
}

// WithNeverExport marks resources that are absent from any export, even when
// explicitly included (ADR-0160).
func WithNeverExport(selectors ...Selector) Option {
	return func(e *Exporter) {
		e.cfg.sel = e.cfg.sel.neverExporting(selectors...)
	}
}

// WithReferences sets how references to resources outside the kept set
// render (ADR-0160). The default [ReferencesOmit] drops them entirely;
// [ReferencesNameOnly] degrades them to plain text names; and
// [ReferencesInclude] pulls one level of pointed-at resources into the
// export.
func WithReferences(mode ReferencesMode) Option {
	return func(e *Exporter) { e.cfg.references = mode }
}

// WithIntroduction sets a markdown introduction rendered at the top of the
// overview, under the generated prose. "[[kind|id]]" references resolve
// against the export's own links; redacted references follow the
// references mode.
func WithIntroduction(markdown string) Option {
	return func(e *Exporter) { e.cfg.introduction = markdown }
}

// WithHide suppresses whole parts of the reference (ADR-0160). The default
// hides nothing; `WithHide()` with no sections is the default.
func WithHide(sections ...HideSection) Option {
	return func(e *Exporter) { e.cfg.hide = visibilityFrom(sections) }
}

// NewExporter creates an exporter with the given options applied in order.
func NewExporter(opts ...Option) *Exporter {
	exp := &Exporter{cfg: config{
		title:       "",
		description: "",
		command:     "",
		logo:        "",
		generatedAt: time.Time{},
		sel:         (*Selection)(nil),
		references:  ReferencesOmit,
		hide: visibility{
			owners:         false,
			repository:     false,
			examples:       false,
			specifications: false,
			docs:           false,
		},
		introduction: "",
	}}

	for _, opt := range opts {
		opt(exp)
	}

	return exp
}

// Export renders the catalog as a self-contained HTML document.
func (e *Exporter) Export(cat *catalog.Catalog) ([]byte, error) {
	out, err := e.render(cat)
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

// ExportTo renders the catalog and writes it to w.
func (e *Exporter) ExportTo(w io.Writer, cat *catalog.Catalog) error {
	out, err := e.render(cat)
	if err != nil {
		return err
	}

	if _, err := io.WriteString(w, out); err != nil {
		return errorfamily.WrapTransient(err, "catalog.architecture.3",
			"architecture: write reference")
	}

	return nil
}

// ExportFile renders the catalog and writes it to path.
func (e *Exporter) ExportFile(cat *catalog.Catalog, path string) error {
	out, err := e.render(cat)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(out), outputPerm); err != nil {
		return errorfamily.WrapInfrastructure(err, "catalog.architecture.4",
			"architecture: write "+path)
	}

	return nil
}

// render resolves defaults, collects the document model and renders it.
func (e *Exporter) render(cat *catalog.Catalog) (string, error) {
	cfg := e.cfg
	if cfg.title == "" {
		cfg.title = string(cat.Title)
	}

	if cfg.generatedAt.IsZero() {
		cfg.generatedAt = time.Now().UTC()
	}

	doc := collect(cat, cfg)

	out, err := renderDocument(doc)
	if err != nil {
		return "", errorfamily.WrapInfrastructure(err, "catalog.architecture.1",
			"architecture: render reference")
	}

	return out, nil
}
