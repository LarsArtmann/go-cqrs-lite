package architecture

import (
	"fmt"
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

// NewExporter creates an exporter with the given options applied in order.
func NewExporter(opts ...Option) *Exporter {
	exp := &Exporter{}

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
			fmt.Sprintf("architecture: write %s", path))
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
